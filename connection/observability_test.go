package connection

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
)

type testObservabilityContext struct {
	collector *har.Collector
	harLevel  logger.LogLevel
}

func (t testObservabilityContext) HARCollector() *har.Collector {
	return t.collector
}

func (t testObservabilityContext) EffectiveHARCollector(_ string, explicit *har.Collector) *har.Collector {
	if explicit != nil {
		return explicit
	}
	if t.harLevel >= logger.Debug {
		return t.collector
	}
	return nil
}

func (t testObservabilityContext) EffectiveHARLevel(_ string) (logger.LogLevel, string) {
	return t.harLevel, "test"
}

func (testObservabilityContext) HTTPLoggingContent(_ string) (bool, bool) {
	return false, false
}

func TestHARDebugCapturesMetadataOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := testObservabilityContext{collector: collector, harLevel: logger.Debug}
	client := &http.Client{Transport: applyHTTPObservability(ctx, "http", http.DefaultTransport, nil)}

	resp, err := client.Post(server.URL+"?q=1", "application/json", strings.NewReader(`{"secret":"value"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	entries := collector.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 HAR entry, got %d", len(entries))
	}
	if entries[0].Request.PostData != nil {
		t.Fatalf("debug HAR should not capture request body: %#v", entries[0].Request.PostData)
	}
	if entries[0].Response.Content.Text != "" {
		t.Fatalf("debug HAR should not capture response body: %q", entries[0].Response.Content.Text)
	}
	if entries[0].Request.HeadersSize != -1 || entries[0].Response.HeadersSize != -1 {
		t.Fatalf("expected metadata header sizes to be unknown")
	}
}

func TestHARTraceCapturesBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := testObservabilityContext{collector: collector, harLevel: logger.Trace}
	client := &http.Client{Transport: applyHTTPObservability(ctx, "http", http.DefaultTransport, nil)}

	resp, err := client.Post(server.URL, "application/json", strings.NewReader(`{"secret":"value"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	entries := collector.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 HAR entry, got %d", len(entries))
	}
	if entries[0].Request.PostData == nil {
		t.Fatalf("trace HAR should capture request body")
	}
	if entries[0].Response.Content.Text == "" {
		t.Fatalf("trace HAR should capture response body")
	}
}
