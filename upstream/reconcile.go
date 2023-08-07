package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/flanksource/commons/logger"
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
func (t *upstreamReconciler) Sync(ctx dbContext, table string) error {
	logger.Debugf("Reconciling table %q with upstream", table)

	var next string
	if table == "check_statuses" {
		next = "," // in the format <check_id>,<time>
	}

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

		logger.Debugf("[table=%s] Pushing %d items to upstream. Next: %s", table, pushData.Count(), next)

		pushData.AgentName = t.upstreamConf.AgentName
		if err := Push(ctx, t.upstreamConf, pushData); err != nil {
			errorList = append(errorList, fmt.Errorf("failed to push missing resource ids: %w", err))
		}
	}

	return errors.Join(errorList...)
}

// fetchUpstreamResourceIDs requests all the existing resource ids from the upstream
// that were sent by this agent.
func (t *upstreamReconciler) fetchUpstreamResourceIDs(ctx dbContext, request PaginateRequest) ([]string, error) {
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
