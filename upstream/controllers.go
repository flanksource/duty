package upstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/attribute"
)

// PullHandler returns a handler that returns all the ids of items it has received from the requested agent.
func PullHandler(allowedTables []string) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)
		var req PaginateRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error()})
		}

		reqJSON, _ := json.Marshal(req)
		ctx.GetSpan().SetAttributes(attribute.String("upstream.pull.paginate-request", string(reqJSON)))

		if !collections.Contains(allowedTables, req.Table) {
			return c.JSON(http.StatusForbidden, api.HTTPError{Error: fmt.Sprintf("table=%s is not allowed", req.Table)})
		}

		agentName := c.Param("agent_name")
		agent, err := query.FindAgent(ctx, agentName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to get agent"})
		} else if agent == nil {
			return c.JSON(http.StatusNotFound, api.HTTPError{Message: fmt.Sprintf("agent(name=%s) not found", agentName)})
		}

		resp, err := query.GetAllResourceIDsOfAgent(ctx, req.Table, req.From, req.Size, agent.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to get resource ids"})
		}

		return c.JSON(http.StatusOK, resp)
	}
}

// PushHandler returns an echo handler that saves the push data from agents.
func PushHandler(agentIDCache *cache.Cache) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)

		var req PushData
		err := json.NewDecoder(c.Request().Body).Decode(&req)
		if err != nil {
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error(), Message: "invalid json request"})
		}

		ctx.GetSpan().SetAttributes(attribute.Int("upstream.push.msg-count", req.Count()))

		req.AgentName = strings.TrimSpace(req.AgentName)
		if req.AgentName == "" {
			return c.JSON(http.StatusBadRequest, api.HTTPError{Error: "agent name is required", Message: "agent name is required"})
		}

		agentID, ok := agentIDCache.Get(req.AgentName)
		if !ok {
			agent, err := GetOrCreateAgent(ctx, req.AgentName)
			if err != nil {
				return c.JSON(http.StatusBadRequest, api.HTTPError{
					Error:   err.Error(),
					Message: "Error while creating/fetching agent",
				})
			}
			agentID = agent.ID
			agentIDCache.Set(req.AgentName, agentID, cache.DefaultExpiration)
		}

		req.PopulateAgentID(agentID.(uuid.UUID))

		logger.Tracef("Inserting push data %s", req.String())
		if err := InsertUpstreamMsg(ctx, &req); err != nil {
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to upsert upstream message"})
		}

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

		reqJSON, _ := json.Marshal(req)
		ctx.GetSpan().SetAttributes(attribute.String("upstream.status.paginate-request", string(reqJSON)))

		if !collections.Contains(allowedTables, req.Table) {
			return c.JSON(http.StatusForbidden, api.HTTPError{Error: fmt.Sprintf("table=%s is not allowed", req.Table)})
		}

		var agentName = c.Param("agent_name")
		agent, err := query.FindAgent(ctx, agentName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to get agent"})
		}

		if agent == nil {
			return c.JSON(http.StatusNotFound, api.HTTPError{Message: fmt.Sprintf("agent(name=%s) not found", agentName)})
		}

		response, err := GetPrimaryKeysHash(ctx, req, agent.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to push status response"})
		}

		return c.JSON(http.StatusOK, response)
	}
}
