package upstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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

func AgentAuthMiddleware(agentCache *cache.Cache) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context().(context.Context)

			// If agent is already set (basic auth) then just proceed
			if ctx.Agent() != nil {
				return next(c)
			}

			histogram := ctx.Histogram("agent_auth_middleware", StatusLabel, "")

			agentName := c.QueryParam(AgentNameQueryParam)
			if agentName == "" {
				histogram.Label(StatusLabel, StatusAgentError)
				return c.JSON(http.StatusBadRequest, api.HTTPError{Error: "agent name is required"})
			}

			var agent *models.Agent
			var err error
			if val, ok := agentCache.Get(agentName); ok {
				agent = val.(*models.Agent)
			} else {
				agent, err = GetOrCreateAgent(ctx, agentName)
				if err != nil {
					histogram.Label(StatusLabel, StatusAgentError)
					return c.JSON(http.StatusBadRequest, api.HTTPError{
						Error: fmt.Errorf("failed to create/fetch agent: %w", err).Error(),
					})
				}

				agentCache.SetDefault(agentName, agent)
			}

			ctx = ctx.WithAgent(*agent)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// PushHandler returns an echo handler that saves the push data from agents.
func PushHandler(c echo.Context) error {
	ctx := c.Request().Context().(context.Context)
	var req PushData
	start := time.Now()
	histogram := ctx.Histogram("push_queue_create_handler", StatusLabel, "", AgentLabel, "")
	defer func() {
		histogram.Since(start)
	}()
	err := json.NewDecoder(c.Request().Body).Decode(&req)
	if err != nil {
		histogram.Label(StatusLabel, StatusAgentError)
		return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error(), Message: "invalid json request"})
	}

	ctx.GetSpan().SetAttributes(attribute.Int("count", req.Count()))

	agentID := ctx.Agent().ID
	histogram = histogram.Label(AgentLabel, agentID.String())
	req.PopulateAgentID(agentID)

	ctx.Tracef("Inserting push data %s", req.String())

	if err := InsertUpstreamMsg(ctx, &req); err != nil {
		histogram.Label(StatusLabel, StatusError)
		return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to upsert upstream message"})
	}
	histogram.Label(StatusLabel, StatusOK)
	req.AddMetrics(ctx.Counter("push_queue_create_handler_records", AgentLabel, agentID.String()))

	if err := UpdateAgentLastReceived(ctx, agentID); err != nil {
		logger.Errorf("failed to update agent last_received: %v", err)
	}

	return nil
}

// PushHandler returns an echo handler that deletes the push data from the upstream.
func DeleteHandler(c echo.Context) error {
	ctx := c.Request().Context().(context.Context)
	start := time.Now()
	var req PushData
	err := json.NewDecoder(c.Request().Body).Decode(&req)
	histogram := ctx.Histogram("push_queue_delete_handler", StatusLabel, "", AgentLabel, "")
	if err != nil {
		histogram.Label(StatusLabel, StatusAgentError).Since(start)
		return c.JSON(http.StatusBadRequest, api.HTTPError{Error: err.Error(), Message: "invalid json request"})
	}

	ctx.GetSpan().SetAttributes(attribute.String("action", "delete"), attribute.Int("upstream.push.msg-count", req.Count()))

	agentID := ctx.Agent().ID
	histogram = histogram.Label(AgentLabel, agentID.String())
	req.PopulateAgentID(agentID)

	ctx.Logger.V(3).Infof("Deleting push data %s", req.String())
	if err := DeleteOnUpstream(ctx, &req); err != nil {
		histogram.Label(StatusLabel, "error").Since(start)
		return c.JSON(http.StatusInternalServerError, api.HTTPError{Error: err.Error(), Message: "failed to upsert upstream message"})
	}

	histogram.Label(StatusLabel, StatusOK).Since(start)
	req.AddMetrics(ctx.Counter("push_queue_delete_handler_records", AgentLabel, agentID.String()))

	if err := UpdateAgentLastReceived(ctx, agentID); err != nil {
		logger.Errorf("failed to update agent last_received: %v", err)
	}

	return nil
}

func PingHandler(c echo.Context) error {
	start := time.Now()
	ctx := c.Request().Context().(context.Context)

	histogram := ctx.Histogram("push_queue_ping_handler", StatusLabel, "", AgentLabel, ctx.Agent().ID.String())

	if err := UpdateAgentLastSeen(ctx, ctx.Agent().ID); err != nil {
		histogram.Label(StatusLabel, StatusError).Since(start)
		return fmt.Errorf("failed to update agent last_seen: %w", err)
	}

	histogram.Label(StatusLabel, StatusOK).Since(start)
	return nil
}
