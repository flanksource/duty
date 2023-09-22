package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	upstreamConf UpstreamConfig

	// the max number of resources the agent fetches
	// from the upstream in one request.
	pageSize int
}

func NewUpstreamReconciler(upstreamConf UpstreamConfig, pageSize int) *upstreamReconciler {
	return &upstreamReconciler{
		upstreamConf: upstreamConf,
		pageSize:     pageSize,
	}
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *upstreamReconciler) Sync(ctx duty.DBContext, table string) error {
	logger.Debugf("Reconciling table %q with upstream", table)

	// Empty starting cursor, so we sync everything
	var next string
	if table == "check_statuses" {
		next = "," // in the format <check_id>,<time>
	}

	return t.sync(ctx, table, next)
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *upstreamReconciler) SyncAfter(ctx duty.DBContext, table string, after time.Duration) error {
	logger.WithValues("since", time.Now().Add(-after).Format(time.RFC3339)).Debugf("Reconciling table %q with upstream", table)

	var next string
	switch table {
	case "check_statuses":
		var checkStatus *models.CheckStatus
		if err := ctx.DB().Select("checks.id as check_id", "check_statuses.time").
			Joins("LEFT JOIN checks ON check_statuses.check_id = checks.id").
			Where("checks.agent_id = ?", uuid.Nil).
			Where("NOW() - check_statuses.created_at <= ?", after).
			Order("check_statuses.created_at").
			First(&checkStatus).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else if checkStatus != nil && checkStatus.CheckID != uuid.Nil && checkStatus.Time != "" {
			next = fmt.Sprintf("%s,%s", checkStatus.CheckID, checkStatus.Time)
		}

	default:
		if err := ctx.DB().Table(table).Select("id").Where("agent_id = ?", uuid.Nil).Where("NOW() - created_at <= ?", after).Order("created_at").Limit(1).Scan(&next).Error; err != nil {
			return err
		}
	}

	if next == "" {
		logger.Debugf("no records found within the given duration")
		return nil
	}

	return t.sync(ctx, table, next)
}

// Sync compares all the resource of the given table against
// the upstream server and pushes any missing resources to the upstream.
func (t *upstreamReconciler) sync(ctx duty.DBContext, table, next string) error {
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
		if err := Push(ctx, t.upstreamConf, pushData); err != nil {
			errorList = append(errorList, fmt.Errorf("failed to push missing resource ids: %w", err))
		}
	}

	return errors.Join(errorList...)
}

// fetchUpstreamResourceIDs requests all the existing resource ids from the upstream
// that were sent by this agent.
func (t *upstreamReconciler) fetchUpstreamResourceIDs(ctx duty.DBContext, request PaginateRequest) ([]string, error) {
	endpoint, err := url.JoinPath(t.upstreamConf.Host, "upstream", "pull", t.upstreamConf.AgentName)
	if err != nil {
		return nil, fmt.Errorf("error creating url endpoint for host %s: %w", t.upstreamConf.Host, err)
	}

	req, err := t.createPaginateRequest(ctx, endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, string(respBody))
	}

	var response []string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}

func (t *upstreamReconciler) fetchUpstreamStatus(ctx context.Context, request PaginateRequest) (*PaginateResponse, error) {
	endpoint, err := url.JoinPath(t.upstreamConf.Host, "upstream", "status", t.upstreamConf.AgentName)
	if err != nil {
		return nil, fmt.Errorf("error creating url endpoint for host %s: %w", t.upstreamConf.Host, err)
	}

	req, err := t.createPaginateRequest(ctx, endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("error creating paginate request: %w", err)
	}

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, string(respBody))
	}

	var response PaginateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (t *upstreamReconciler) createPaginateRequest(ctx context.Context, url string, request PaginateRequest) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}

	query := req.URL.Query()
	query.Add("table", request.Table)
	query.Add("from", request.From)
	query.Add("size", fmt.Sprintf("%d", request.Size))
	req.URL.RawQuery = query.Encode()

	req.SetBasicAuth(t.upstreamConf.Username, t.upstreamConf.Password)
	return req, nil
}
