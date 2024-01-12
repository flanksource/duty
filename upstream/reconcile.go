package upstream

import (
	gocontext "context"
	"encoding/json"
	"errors"
	"fmt"
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

func (p PaginateRequest) String() string {
	return fmt.Sprintf("table=%s next=%s, size=%d", p.Table, p.From, p.Size)
}

func (p PaginateResponse) String() string {
	return fmt.Sprintf("hash=%s, next=%s, count=%d", p.Hash, p.Next, p.Total)
}

// UpstreamReconciler pushes missing resources from an agent to the upstream.
type UpstreamReconciler struct {
	upstreamConf   UpstreamConfig
	upstreamClient *UpstreamClient

	// the max number of resources the agent fetches
	// from the upstream in one request.
	pageSize int
}

func NewUpstreamReconciler(upstreamConf UpstreamConfig, pageSize int) *UpstreamReconciler {
	return &UpstreamReconciler{
		upstreamConf:   upstreamConf,
		pageSize:       pageSize,
		upstreamClient: NewUpstreamClient(upstreamConf),
	}
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *UpstreamReconciler) Sync(ctx context.Context, table string) (int, error) {
	logger.Debugf("Reconciling table %q with upstream", table)

	// Empty starting cursor, so we sync everything
	return t.sync(ctx, table, uuid.Nil.String())
}

// SyncAfter pushes all the records of the given table that were updated in the given duration
func (t *UpstreamReconciler) SyncAfter(ctx context.Context, table string, after time.Duration) (int, error) {
	logger.WithValues("since", time.Now().Add(-after).Format(time.RFC3339Nano)).Debugf("Reconciling table %q with upstream", table)

	// We find the item that falls just before the requested duration & begin from there
	//var next string
	//if err := ctx.DB().Table(table).Select("id").Where("agent_id = ?", uuid.Nil).Where("NOW() - updated_at > ?", after).Order("id").Limit(1).Scan(&next).Error; err != nil {
	//return err
	//}
	//if err := ctx.DB().Table(table).Select("id").Where("agent_id = ?", uuid.Nil).Where("NOW() - updated_at > ?", after).Order("id").Limit(1).Scan(&next).Error; err != nil {
	//return err
	//}

	// We start with a nil UUID and calculate hash in batches
	next := uuid.Nil.String()
	return t.sync(ctx, table, next)
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *UpstreamReconciler) sync(ctx context.Context, table, next string) (int, error) {
	var errorList []error
	// We keep this counter to keep a track of attempts for a batch
	pushed := 0
	for {
		paginateRequest := PaginateRequest{From: next, Table: table, Size: t.pageSize}

		localStatus, err := GetPrimaryKeysHash(ctx, paginateRequest, uuid.Nil)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch hash of primary keys from local db: %w", err)
		}

		// Nothing left to push
		if localStatus.Total == 0 {
			break
		}

		if localStatus.Hash == "" {
			return 0, fmt.Errorf("empty row hash returned")
		}

		upstreamStatus, err := t.fetchUpstreamStatus(ctx, paginateRequest)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch upstream status: %w", err)
		}

		next = localStatus.Next

		if upstreamStatus.Hash == localStatus.Hash {
			logger.Debugf("[%s] pages matched,  local(%s) == upstream(%s)", paginateRequest, localStatus, upstreamStatus)
			continue
		}
		logger.Debugf("[%s] local(%s) == upstream(%s)", paginateRequest, localStatus, upstreamStatus)

		resp, err := t.fetchUpstreamResourceIDs(ctx, paginateRequest)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch upstream resource ids: %w", err)
		}

		pushData, err := GetMissingResourceIDs(ctx, resp, paginateRequest)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch missing resource ids: %w", err)
		}

		if pushData != nil && pushData.Count() > 0 {
			logger.WithValues("table", table).Debugf("Pushing %d items to upstream. Next: %q", pushData.Count(), next)

			pushData.AgentName = t.upstreamConf.AgentName
			if err := t.upstreamClient.Push(ctx, pushData); err != nil {
				errorList = append(errorList, fmt.Errorf("failed to push missing resource ids: %w", err))
			}
			pushed += pushData.Length()
		}
		if next == "" {
			break
		}
	}

	return pushed, errors.Join(errorList...)
}

// fetchUpstreamResourceIDs requests all the existing resource ids from the upstream
// that were sent by this agent.
func (t *UpstreamReconciler) fetchUpstreamResourceIDs(ctx context.Context, request PaginateRequest) ([]string, error) {
	httpReq := t.createPaginateRequest(ctx, request)
	httpResponse, err := httpReq.Get(fmt.Sprintf("pull/%s", t.upstreamConf.AgentName))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	body, err := httpResponse.AsString()
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if !httpResponse.IsOK() {
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", httpResponse.StatusCode, parseResponse(body))
	}

	var response []string
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return nil, fmt.Errorf("invalid response format: %s", parseResponse(body))
	}

	return response, nil
}

func (t *UpstreamReconciler) fetchUpstreamStatus(ctx gocontext.Context, request PaginateRequest) (*PaginateResponse, error) {
	httpReq := t.createPaginateRequest(ctx, request)
	httpResponse, err := httpReq.Get(fmt.Sprintf("status/%s", t.upstreamConf.AgentName))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	body, err := httpResponse.AsString()
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if !httpResponse.IsOK() {
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", httpResponse.StatusCode, parseResponse(body))
	}

	var response PaginateResponse
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return nil, fmt.Errorf("invalid response format: %s: %v", parseResponse(body), err)
	}

	return &response, nil
}

func (t *UpstreamReconciler) createPaginateRequest(ctx gocontext.Context, request PaginateRequest) *http.Request {
	return t.upstreamClient.R(ctx).
		QueryParam("table", request.Table).
		QueryParam("from", request.From).
		QueryParam("size", fmt.Sprintf("%d", request.Size))
}

func parseResponse(body string) string {
	if len(body) > 200 {
		body = body[0:200]
	}
	return body
}
