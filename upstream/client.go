package upstream

import (
	"context"
	"fmt"
	"io"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"go.opentelemetry.io/otel"
)

type UpstreamClient struct {
	httpClient *http.Client
}

func NewUpstreamClient(config UpstreamConfig) *UpstreamClient {
	tracedTransport := middlewares.NewTracedTransport().
		TraceProvider(otel.GetTracerProvider()).
		TraceAll(true).
		MaxBodyLength(512)

	return &UpstreamClient{
		httpClient: http.NewClient().
			Auth(config.Username, config.Password).
			BaseURL(fmt.Sprintf("%s/upstream", config.Host)).
			Use(tracedTransport.RoundTripper),
	}
}

// Push uploads the given push message to the upstream server.
func (t *UpstreamClient) Push(ctx context.Context, msg *PushData) error {
	if msg.Count() == 0 {
		return nil
	}

	resp, err := t.httpClient.R(ctx).Post("push", msg)
	if err != nil {
		return fmt.Errorf("error pushing to upstream: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsOK() {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream server returned error status[%d]: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
