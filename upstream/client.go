package upstream

import (
	"fmt"
	"io"
	netHTTP "net/http"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
)

type UpstreamClient struct {
	AgentName string
	*http.Client
}

func NewUpstreamClient(config UpstreamConfig) *UpstreamClient {
	client := UpstreamClient{
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
	resp, err := t.Client.R(ctx).QueryParam("agent_name", t.AgentName).Get("/ping")
	if !resp.IsOK() {
		return fmt.Errorf("upstream sent an unexpected response: %v", resp.StatusCode)
	}

	return err
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
	msg.AddMetrics(ctx.Counter("push_queue_records", "method", method, "agent", msg.AgentName))
	histogram := ctx.Histogram("push_queue_batch", "method", method, "agent", msg.AgentName)
	req := t.R(ctx).QueryParam("agent_name", msg.AgentName)
	if err := req.Body(msg); err != nil {
		return fmt.Errorf("error setting body: %w", err)
	}

	resp, err := req.Do(method, "push")
	if err != nil {
		histogram.Label(StatusLabel, StatusError).Since(start)
		return fmt.Errorf("error pushing to upstream: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsOK() {
		histogram.Label(StatusLabel, StatusError).Since(start)
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, parseResponse(string(respBody)))
	}
	histogram.Label(StatusLabel, StatusOK).Since(start)
	return nil
}
