package upstream

import (
	gocontext "context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
)

type PaginateRequest struct {
	Table string `query:"table"`
	From  string `query:"from"`
	Size  int    `query:"size"`
}

type PaginateResponse struct {
	Hash  string `gorm:"column:sha256sum"`
	Next  string `gorm:"column:last_id"`
	Total int    `gorm:"column:total"`
}

// upstreamReconciler pushes missing resources from an agent to the upstream.
type upstreamReconciler struct {
	upstreamConf   UpstreamConfig
	upstreamClient *UpstreamClient

	// the max number of resources the agent fetches
	// from the upstream in one request.
	pageSize int
}

func NewUpstreamReconciler(upstreamConf UpstreamConfig, pageSize int) *upstreamReconciler {
	return &upstreamReconciler{
		upstreamConf:   upstreamConf,
		pageSize:       pageSize,
		upstreamClient: NewUpstreamClient(upstreamConf),
	}
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *upstreamReconciler) Sync(ctx context.Context, table string) error {
	logger.Debugf("Reconciling table %q with upstream", table)

	// Empty starting cursor, so we sync everything
	return t.sync(ctx, table, "")
}

// SyncAfter pushes all the records of the given table that were updated in the given duration
func (t *upstreamReconciler) SyncAfter(ctx context.Context, table string, after time.Duration) error {
	logger.WithValues("since", time.Now().Add(-after).Format(time.RFC3339Nano)).Debugf("Reconciling table %q with upstream", table)

	// We find the item that falls just before the requested duration & begin from there.
	var next string
	if err := ctx.DB().Table(table).Select("id").Where("agent_id = ?", uuid.Nil).Where("NOW() - updated_at > ?", after).Order("updated_at DESC").Limit(1).Scan(&next).Error; err != nil {
		return err
	}

	return t.sync(ctx, table, next)
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *upstreamReconciler) sync(ctx context.Context, table, next string) error {
	var errorList []error
	for {
		paginateRequest := PaginateRequest{From: next, Table: table, Size: t.pageSize}

		localStatus, err := GetPrimaryKeysHash(ctx, paginateRequest, uuid.Nil)
		if err != nil {
			return fmt.Errorf("failed to fetch hash of primary keys from local db: %w", err)
		}
		next = localStatus.Next

		// Nothing left to push
		if localStatus.Total == 0 {
			break
		}

		upstreamStatus, err := t.fetchUpstreamStatus(ctx, paginateRequest)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream status: %w", err)
		}

		if upstreamStatus.Hash == localStatus.Hash {
			continue
		}

		resp, err := t.fetchUpstreamResourceIDs(ctx, paginateRequest)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream resource ids: %w", err)
		}

		pushData, err := GetMissingResourceIDs(ctx, resp, paginateRequest)
		if err != nil {
			return fmt.Errorf("failed to fetch missing resource ids: %w", err)
		}

		logger.WithValues("table", table).Debugf("Pushing %d items to upstream. Next: %q", pushData.Count(), next)

		pushData.AgentName = t.upstreamConf.AgentName
		if err := t.upstreamClient.Push(ctx, pushData); err != nil {
			errorList = append(errorList, fmt.Errorf("failed to push missing resource ids: %w", err))
		}
	}

	return errors.Join(errorList...)
}

// fetchUpstreamResourceIDs requests all the existing resource ids from the upstream
// that were sent by this agent.
func (t *upstreamReconciler) fetchUpstreamResourceIDs(ctx context.Context, request PaginateRequest) ([]string, error) {
	httpReq := t.createPaginateRequest(ctx, request)
	httpResponse, err := httpReq.Get(fmt.Sprintf("pull/%s", t.upstreamConf.AgentName))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer httpResponse.Body.Close()

	if !httpResponse.IsOK() {
		respBody, _ := io.ReadAll(httpResponse.Body)
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", httpResponse.StatusCode, string(respBody))
	}

	var response []string
	if err := httpResponse.Into(&response); err != nil {
		return nil, err
	}

	return response, nil
}

func (t *upstreamReconciler) fetchUpstreamStatus(ctx gocontext.Context, request PaginateRequest) (*PaginateResponse, error) {
	httpReq := t.createPaginateRequest(ctx, request)
	httpResponse, err := httpReq.Get(fmt.Sprintf("status/%s", t.upstreamConf.AgentName))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer httpResponse.Body.Close()

	if !httpResponse.IsOK() {
		respBody, _ := io.ReadAll(httpResponse.Body)
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", httpResponse.StatusCode, string(respBody))
	}

	var response PaginateResponse
	if err := httpResponse.Into(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (t *upstreamReconciler) createPaginateRequest(ctx gocontext.Context, request PaginateRequest) *http.Request {
	return t.upstreamClient.R(ctx).
		QueryParam("table", request.Table).
		QueryParam("from", request.From).
		QueryParam("size", fmt.Sprintf("%d", request.Size))
}
