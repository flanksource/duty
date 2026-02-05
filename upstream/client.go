package upstream

import (
	"encoding/json"
	"fmt"
	"io"
	netHTTP "net/http"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	gocache "github.com/patrickmn/go-cache"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
)

// AgentNameQueryParam is the name of the query param that's used to authenticate an
// agent when using basic auth instead of access tokens.
const AgentNameQueryParam = "agent_name"

type UpstreamClient struct {
	*http.Client
	AgentName string

	// cache upstream records
	cache *gocache.Cache
}

func NewUpstreamClient(config UpstreamConfig) *UpstreamClient {
	client := UpstreamClient{
		cache:     gocache.New(5*time.Minute, 10*time.Minute),
		AgentName: config.AgentName,
		Client: http.NewClient().
			Auth(config.Username, config.Password).
			InsecureSkipVerify(config.InsecureSkipVerify).
			BaseURL(fmt.Sprintf("%s/upstream", config.Host)).
			Trace(http.TraceConfig{
				QueryParam: true,
			}),
	}
	for _, opt := range config.Options {
		opt(client.Client)
	}
	return &client
}

// PushArtifacts uploads the given artifact to the upstream server.
func (t *UpstreamClient) PushArtifacts(ctx context.Context, artifactID uuid.UUID, reader io.ReadCloser) error {
	resp, err := t.R(ctx).Post(fmt.Sprintf("artifacts/%s", artifactID), reader)
	if err != nil {
		return fmt.Errorf("error pushing to upstream: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsOK() {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, parseResponse(string(respBody)))
	}

	return nil
}

// Ping sends a ping message to the upstream
func (t *UpstreamClient) Ping(ctx context.Context) error {
	resp, err := t.Client.R(ctx).QueryParam(AgentNameQueryParam, t.AgentName).Get("/ping")
	if err != nil {
		return err
	}

	if !resp.IsOK() {
		return fmt.Errorf("upstream sent an unexpected response: %v", resp.StatusCode)
	}

	return nil
}

// Push uploads the given push message to the upstream server.
func (t *UpstreamClient) Push(ctx context.Context, msg *PushData) error {
	return t.push(ctx, netHTTP.MethodPost, msg)
}

// Delete performs hard delete on the given items from the upstream server.
func (t *UpstreamClient) Delete(ctx context.Context, msg *PushData) error {
	return t.push(ctx, netHTTP.MethodDelete, msg)
}

func (t *UpstreamClient) push(ctx context.Context, method string, msg *PushData) error {
	if msg.Count() == 0 {
		return nil
	}

	start := time.Now()
	msg.AddMetrics(ctx.Counter("push_queue_records", "method", method, "agent", t.AgentName, "table", ""))
	histogram := ctx.Histogram("push_queue_batch", context.LatencyBuckets, "method", method, "agent", t.AgentName, StatusLabel, "")
	req := t.R(ctx).QueryParam(AgentNameQueryParam, t.AgentName)
	if err := req.Body(msg); err != nil {
		return fmt.Errorf("error setting body: %w", err)
	}

	resp, err := req.Do(method, "push")
	if err != nil {
		histogram.Label(StatusLabel, StatusError).Since(start)
		msgSize, msgSizeErr := msg.Size()
		if msgSizeErr != nil {
			logger.Errorf("failed to get msgsize: %w", msgSizeErr)
		}

		return fmt.Errorf("error pushing to upstream (msg_size: %d bytes): %w", msgSize, err)
	}
	defer resp.Body.Close()

	if !resp.IsOK() {
		histogram.Label(StatusLabel, StatusError).Since(start)
		respBody, _ := io.ReadAll(resp.Body)

		var httpErr api.HTTPError
		if json.Unmarshal(respBody, &httpErr) == nil {
			return &httpErr
		}
		return fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, parseResponse(string(respBody)))
	}
	histogram.Label(StatusLabel, StatusOK).Since(start)
	return nil
}

// ListViews returns all views from upstream with namespace,name pairs
func (t *UpstreamClient) ListViews(ctx context.Context, views []ViewIdentifier) ([]ViewWithColumns, error) {
	cacheKey := "columns-list"

	if cached, found := t.cache.Get(cacheKey); found {
		return cached.([]ViewWithColumns), nil
	}

	req := t.R(ctx).QueryParam(AgentNameQueryParam, t.AgentName)
	if err := req.Body(views); err != nil {
		return nil, fmt.Errorf("error setting request body: %w", err)
	}

	resp, err := req.Do(netHTTP.MethodPost, "list-views")
	if err != nil {
		return nil, fmt.Errorf("error listing views on upstream: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsOK() {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, parseResponse(string(respBody)))
	}

	var result []ViewWithColumns
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	t.cache.Set(cacheKey, result, ctx.Properties().Duration("upstream.client.cache.view-columns.duration", gocache.DefaultExpiration))
	return result, nil
}

func parseResponse(body string) string {
	if len(body) > 200 {
		body = body[0:200]
	}
	return body
}
