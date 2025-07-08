package upstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/attribute"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/view"
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

			histogram := ctx.Histogram("agent_auth_middleware", context.ShortLatencyBuckets, StatusLabel, "")

			agentName := c.QueryParam(AgentNameQueryParam)
			if agentName == "" {
				histogram.Label(StatusLabel, StatusAgentError)
				return c.JSON(http.StatusBadRequest, api.HTTPError{Err: "agent name is required"})
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
						Err: fmt.Errorf("failed to create/fetch agent: %w", err).Error(),
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

// NewPushHandler returns an echo handler that saves the push data from agents.
func NewPushHandler(ringManager StatusRingManager) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context().(context.Context)

		start := time.Now()
		histogram := ctx.Histogram("push_queue_create_handler", context.LatencyBuckets, StatusLabel, "", AgentLabel, "")
		defer func() {
			histogram.Since(start)
		}()

		var req PushData
		err := json.NewDecoder(c.Request().Body).Decode(&req)
		if err != nil {
			histogram.Label(StatusLabel, StatusAgentError)
			return c.JSON(http.StatusBadRequest, api.HTTPError{Err: err.Error(), Message: "invalid json request"})
		}

		ctx.GetSpan().SetAttributes(attribute.Int("count", req.Count()))

		agentID := ctx.Agent().ID
		histogram = histogram.Label(AgentLabel, agentID.String())
		req.PopulateAgentID(agentID)

		ctx.Logger.V(6).Infof("inserting push data %s", req.String())

		if err := InsertUpstreamMsg(ctx, &req); err != nil {
			histogram.Label(StatusLabel, StatusError)
			return api.WriteError(c, err)
		}

		addJobHistoryToRing(ctx, agentID.String(), req.JobHistory, ringManager)

		histogram.Label(StatusLabel, StatusOK)
		req.AddMetrics(ctx.Counter("push_queue_create_handler_records", AgentLabel, agentID.String(), "table", ""))

		if err := UpdateAgentLastReceived(ctx, agentID); err != nil {
			logger.Errorf("failed to update agent last_received: %v", err)
		}

		return nil
	}
}

func addJobHistoryToRing(ctx context.Context, agentID string, histories []models.JobHistory, ringManager StatusRingManager) {
	if ringManager == nil {
		return
	}
	job.StartJobHistoryEvictor(ctx)

	for _, history := range histories {
		ringManager.Add(ctx, agentID, history)
	}
}

// PushHandler returns an echo handler that deletes the push data from the upstream.
func DeleteHandler(c echo.Context) error {
	ctx := c.Request().Context().(context.Context)
	start := time.Now()
	var req PushData
	err := json.NewDecoder(c.Request().Body).Decode(&req)
	histogram := ctx.Histogram("push_queue_delete_handler", context.LatencyBuckets, StatusLabel, "", AgentLabel, "")
	if err != nil {
		histogram.Label(StatusLabel, StatusAgentError).Since(start)
		return c.JSON(http.StatusBadRequest, api.HTTPError{Err: err.Error(), Message: "invalid json request"})
	}

	ctx.GetSpan().SetAttributes(attribute.String("action", "delete"), attribute.Int("upstream.push.msg-count", req.Count()))

	agentID := ctx.Agent().ID
	histogram = histogram.Label(AgentLabel, agentID.String())
	req.PopulateAgentID(agentID)

	ctx.Logger.V(3).Infof("Deleting push data %s", req.String())
	if err := DeleteOnUpstream(ctx, &req); err != nil {
		histogram.Label(StatusLabel, "error").Since(start)
		return c.JSON(http.StatusInternalServerError, api.HTTPError{Err: err.Error(), Message: "failed to delete items"})
	}

	histogram.Label(StatusLabel, StatusOK).Since(start)
	req.AddMetrics(ctx.Counter("push_queue_delete_handler_records", AgentLabel, agentID.String(), "table", ""))

	if err := UpdateAgentLastReceived(ctx, agentID); err != nil {
		logger.Errorf("failed to update agent last_received: %v", err)
	}

	return nil
}

func PingHandler(c echo.Context) error {
	start := time.Now()
	ctx := c.Request().Context().(context.Context)

	histogram := ctx.Histogram("push_queue_ping_handler", context.ShortLatencyBuckets, StatusLabel, "", AgentLabel, ctx.Agent().ID.String())

	if err := UpdateAgentLastSeen(ctx, ctx.Agent().ID); err != nil {
		histogram.Label(StatusLabel, StatusError).Since(start)
		return fmt.Errorf("failed to update agent last_seen: %w", err)
	}

	histogram.Label(StatusLabel, StatusOK).Since(start)
	return nil
}

// CheckViewHandler checks if a view with the same namespace, name and column definition exists
func CheckViewHandler(c echo.Context) error {
	ctx := c.Request().Context().(context.Context)

	var req struct {
		Namespace string               `json:"namespace"`
		Name      string               `json:"name"`
		Columns   []view.ViewColumnDef `json:"columns"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, api.HTTPError{Err: err.Error(), Message: "invalid json request"})
	}

	var viewModel models.View
	if err := ctx.DB().Where("namespace = ? AND name = ?", req.Namespace, req.Name).Find(&viewModel).Error; err != nil {
		return c.JSON(http.StatusNotFound, api.HTTPError{Err: err.Error(), Message: "view not found"})
	} else if viewModel.ID == uuid.Nil {
		return c.JSON(http.StatusNotFound, api.HTTPError{Err: "view not found", Message: "view not found"})
	}

	if !ctx.DB().Migrator().HasTable(viewModel.GeneratedTableName()) {
		return c.JSON(http.StatusOK, map[string]string{"status": "found"})
	}

	colDefs, err := view.GetViewColumnDefs(ctx, req.Namespace, req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.HTTPError{Err: err.Error(), Message: "failed to get view column defs"})
	}

	// Check if column definitions match
	if len(colDefs) != len(req.Columns) {
		return c.JSON(http.StatusConflict, api.HTTPError{Message: "column count mismatch"})
	}

	for i, reqCol := range req.Columns {
		if reqCol.Name != colDefs[i].Name || reqCol.Type != colDefs[i].Type {
			return c.JSON(http.StatusConflict, api.HTTPError{Message: "column definition mismatch"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "found"})
}
