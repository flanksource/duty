package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
)

type PaginateRequest struct {
	Table string    `query:"table"`
	From  uuid.UUID `query:"from"`
	Size  int       `query:"size"`
}

type PaginateResponse struct {
	Hash  string    `gorm:"column:sha256sum"`
	Next  uuid.UUID `gorm:"column:last_id"`
	Total int       `gorm:"column:total"`
}

type upstreamSyncer struct {
	upstreamConf UpstreamConfig

	// the max number of resources the agent fetches
	// from the upstream in one request.
	pageSize int
}

func NewUpstreamSyncer(upstreamConf UpstreamConfig, pageSize int) *upstreamSyncer {
	return &upstreamSyncer{
		upstreamConf: upstreamConf,
		pageSize:     pageSize,
	}
}

func (t *upstreamSyncer) SyncTableWithUpstream(ctx dbContext, table string) error {
	logger.Infof("Syncing table %q with upstream", table)

	var next uuid.UUID
	for {
		paginateRequest := PaginateRequest{From: next, Table: table, Size: t.pageSize}

		current, err := GetIDsHash(ctx, table, next, t.pageSize)
		if err != nil {
			return fmt.Errorf("failed to fetch local id hash: %w", err)
		}
		next = current.Next

		if current.Total == 0 {
			break
		}

		upstreamStatus, err := t.fetchUpstreamStatus(ctx, paginateRequest)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream status: %w", err)
		}

		if upstreamStatus.Hash == current.Hash {
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

		pushData.AgentName = t.upstreamConf.AgentName
		if err := Push(ctx, t.upstreamConf, pushData); err != nil {
			return fmt.Errorf("failed to push missing resource ids: %w", err)
		}
	}

	return nil
}

// fetchUpstreamResourceIDs requests all the existing resource ids from the upstream
// that were sent by this agent.
func (t *upstreamSyncer) fetchUpstreamResourceIDs(ctx dbContext, request PaginateRequest) ([]string, error) {
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

func (t *upstreamSyncer) fetchUpstreamStatus(ctx context.Context, request PaginateRequest) (*PaginateResponse, error) {
	endpoint, err := url.JoinPath(t.upstreamConf.Host, "upstream", "status", t.upstreamConf.AgentName)
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

func (t *upstreamSyncer) createPaginateRequest(ctx context.Context, url string, request PaginateRequest) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}

	query := req.URL.Query()
	query.Add("table", request.Table)
	query.Add("from", request.From.String())
	query.Add("size", fmt.Sprintf("%d", request.Size))
	req.URL.RawQuery = query.Encode()

	req.SetBasicAuth(t.upstreamConf.Username, t.upstreamConf.Password)
	return req, nil
}
