package upstream

import (
	"context"
	"fmt"
	"io"

	"github.com/flanksource/commons/http"
)

type UpstreamClient struct {
	*http.Client
}

func NewUpstreamClient(config UpstreamConfig) *UpstreamClient {
	client := UpstreamClient{
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

// Push uploads the given push message to the upstream server.
func (t *UpstreamClient) Push(ctx context.Context, msg *PushData) error {
	if msg.Count() == 0 {
		return nil
	}

	resp, err := t.R(ctx).Post("push", msg)
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
