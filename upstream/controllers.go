package upstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/attribute"
)

const (
	StatusAgentError = "agent-error"
	StatusError      = "error"
	StatusOK         = "ok"
	StatusLabel      = "status"
	AgentLabel       = "agent"
)

// PullHandler returns a handler that returns all the ids of items it has received from the requested agent.
func PullHandler(allowedTables []string) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)
		histogram := ctx.Histogram("push_queue_pull_handler")
		start := time.Now()
		defer func() {
			histogram.Since(start)
		}()
		var req PaginateRequest
		if err := c.Bind(&req); err != nil {
			histogram = histogram.Label(StatusLabel, StatusError)
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error()})
		}

		ctx.GetSpan().SetAttributes(
			attribute.String("request.table", req.Table),
			attribute.String("request.from", req.From),
			attribute.Int("request.size", req.Size),
		)

		if !collections.Contains(allowedTables, req.Table) {
			histogram = histogram.Label(StatusLabel, StatusError)
			return c.JSON(http.StatusForbidden, api.HTTPError{Error: fmt.Sprintf("table=%s is not allowed", req.Table)})
		}

		agentName := c.Param("agent_name")
		agent, err := query.FindAgent(ctx, agentName)
		if err != nil {
			histogram = histogram.Label(StatusLabel, StatusError)
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to get agent"})
		} else if agent == nil {
			histogram = histogram.Label(StatusLabel, StatusAgentError)
			return c.JSON(http.StatusNotFound, api.HTTPError{Message: fmt.Sprintf("agent(name=%s) not found", agentName)})
		}
		histogram = histogram.Label(AgentLabel, agent.ID.String())

		resp, err := query.GetAllResourceIDsOfAgent(ctx, req.Table, req.From, req.Size, agent.ID)
		if err != nil {
			histogram.Label(StatusLabel, StatusError)
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to get resource ids"})
		}
		histogram.Label(StatusLabel, StatusOK)
		ctx.Counter("push_queue_pull_handler_records", AgentLabel, agent.ID.String()).Add(len(resp))
		ctx.GetSpan().SetAttributes(attribute.Int("response.count", len(resp)))

		return c.JSON(http.StatusOK, resp)
	}
}

// PushHandler returns an echo handler that saves the push data from agents.
func PushHandler(agentIDCache *cache.Cache) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)
		var req PushData
		start := time.Now()
		histogram := ctx.Histogram("push_queue_create_handler")
		defer func() {
			histogram.Since(start)
		}()
		err := json.NewDecoder(c.Request().Body).Decode(&req)
		if err != nil {
			histogram.Label(StatusLabel, StatusAgentError)
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error(), Message: "invalid json request"})
		}

		ctx.GetSpan().SetAttributes(attribute.Int("count", req.Count()))

		req.AgentName = strings.TrimSpace(req.AgentName)
		if req.AgentName == "" {
			histogram.Label(StatusLabel, StatusAgentError)
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: "agent name is required", Message: "agent name is required"})
		}

		agentID, ok := agentIDCache.Get(req.AgentName)
		if !ok {
			agent, err := GetOrCreateAgent(ctx, req.AgentName)
			if err != nil {
				histogram.Label(StatusLabel, StatusAgentError)
				return c.JSON(http.StatusBadRequest, api.HTTPError{
					Error:   err.Error(),
					Message: "Error while creating/fetching agent",
				})
			}
			agentID = agent.ID
			agentIDCache.Set(req.AgentName, agentID, cache.DefaultExpiration)
		}

		histogram = histogram.Label(AgentLabel, agentID.(uuid.UUID).String())
		req.PopulateAgentID(agentID.(uuid.UUID))

		ctx.Tracef("Inserting push data %s", req.String())

		if err := InsertUpstreamMsg(ctx, &req); err != nil {
			histogram.Label(StatusLabel, StatusError)
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to upsert upstream message"})
		}
		histogram.Label(StatusLabel, StatusOK)
		req.AddMetrics(ctx.Counter("push_queue_create_handler_records", AgentLabel, agentID.(uuid.UUID).String()))

		return nil
	}
}

// PushHandler returns an echo handler that deletes the push data from the upstream.
func DeleteHandler(agentIDCache *cache.Cache) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)
		start := time.Now()
		var req PushData
		err := json.NewDecoder(c.Request().Body).Decode(&req)
		histogram := ctx.Histogram("push_queue_delete_handler")
		if err != nil {
			histogram.Label(StatusLabel, StatusAgentError).Since(start)
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error(), Message: "invalid json request"})
		}

		ctx.GetSpan().SetAttributes(attribute.String("action", "delete"), attribute.Int("upstream.push.msg-count", req.Count()))

		req.AgentName = strings.TrimSpace(req.AgentName)
		if req.AgentName == "" {
			histogram.Label(StatusLabel, StatusAgentError).Since(start)
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: "agent name is required", Message: "agent name is required"})
		}

		agentID, ok := agentIDCache.Get(req.AgentName)

		if !ok {
			agent, err := GetOrCreateAgent(ctx, req.AgentName)
			if err != nil {
				histogram.Label(StatusLabel, StatusAgentError).Since(start)
				return c.JSON(http.StatusBadRequest, api.HTTPError{
					Error:   err.Error(),
					Message: "Error while creating/fetching agent",
				})
			}
			agentID = agent.ID
			agentIDCache.Set(req.AgentName, agentID, cache.DefaultExpiration)
		}
		histogram = histogram.Label(AgentLabel, agentID.(uuid.UUID).String())
		req.PopulateAgentID(agentID.(uuid.UUID))

		ctx.Logger.V(3).Infof("Deleting push data %s", req.String())
		if err := DeleteOnUpstream(ctx, &req); err != nil {
			histogram.Label(StatusLabel, "error").Since(start)
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to upsert upstream message"})
		}

		histogram.Label(StatusLabel, StatusOK).Since(start)
		req.AddMetrics(ctx.Counter("push_queue_delete_handler_records", AgentLabel, agentID.(uuid.UUID).String()))
		return nil
	}
}

// StatusHandler returns a handler that returns the summary of all ids the upstream has received.
func StatusHandler(allowedTables []string) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)
		var req PaginateRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error()})
		}
		start := time.Now()
		ctx.GetSpan().SetAttributes(
			attribute.String("request.table", req.Table),
			attribute.String("request.from", req.From),
			attribute.Int("request.size", req.Size),
		)
		if !collections.Contains(allowedTables, req.Table) {
			return c.JSON(http.StatusForbidden, api.HTTPError{Error: fmt.Sprintf("table=%s is not allowed", req.Table)})
		}

		var agentName = c.Param("agent_name")
		histogram := ctx.Histogram("push_queue_status_handler")
		agent, err := query.FindAgent(ctx, agentName)
		if err != nil {
			histogram.Label(StatusLabel, StatusAgentError).Since(start)
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to get agent"})
		}

		if agent == nil {
			histogram.Label(StatusLabel, StatusAgentError).Since(start)
			return c.JSON(http.StatusNotFound, api.HTTPError{Message: fmt.Sprintf("agent(name=%s) not found", agentName)})
		}
		histogram = histogram.Label(AgentLabel, agent.ID.String())

		response, err := GetPrimaryKeysHash(ctx, req, agent.ID)
		if err != nil {
			histogram.Label(StatusLabel, StatusError).Since(start)
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to push status response"})
		}

		histogram.Label(StatusLabel, StatusOK).Since(start)
		return c.JSON(http.StatusOK, response)
	}
}
