package connection

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
	"github.com/onsi/gomega"
)

type testObservabilityContext struct {
	collector  *har.Collector
	harLevel   logger.LogLevel
	logHeaders bool
	logBodies  bool
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

func (t testObservabilityContext) HTTPLoggingContent(_ string) (bool, bool) {
	return t.logHeaders, t.logBodies
}

func TestHARDebugCapturesMetadataOnly(t *testing.T) {
	g := gomega.NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := testObservabilityContext{collector: collector, harLevel: logger.Debug}
	client := &http.Client{Transport: applyHTTPObservability(ctx, "http", http.DefaultTransport, nil)}

	resp, err := client.Post(server.URL+"?q=1", "application/json", strings.NewReader(`{"secret":"value"}`))
	g.Expect(err).ToNot(gomega.HaveOccurred())
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	entries := collector.Entries()
	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Request.PostData).To(gomega.BeNil(), "debug HAR should not capture request body")
	g.Expect(entries[0].Response.Content.Text).To(gomega.BeEmpty(), "debug HAR should not capture response body")
	g.Expect(entries[0].Request.HeadersSize).To(gomega.Equal(-1))
	g.Expect(entries[0].Response.HeadersSize).To(gomega.Equal(-1))
}

func TestHARTraceCapturesBodies(t *testing.T) {
	g := gomega.NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := testObservabilityContext{collector: collector, harLevel: logger.Trace}
	client := &http.Client{Transport: applyHTTPObservability(ctx, "http", http.DefaultTransport, nil)}

	resp, err := client.Post(server.URL, "application/json", strings.NewReader(`{"secret":"value"}`))
	g.Expect(err).ToNot(gomega.HaveOccurred())
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	entries := collector.Entries()
	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Request.PostData).ToNot(gomega.BeNil(), "trace HAR should capture request body")
	g.Expect(entries[0].Response.Content.Text).ToNot(gomega.BeEmpty(), "trace HAR should capture response body")
}

// TestHARAndHTTPLoggingBodiesCoexist exercises both middlewares stacked: HAR
// trace-level body capture AND httpretty-style HTTP body logging. Each layer
// reads and must restore the request/response bodies for the next layer.
// Without correct body restoration, one of three observers would see an empty
// body: the server (downstream of both middlewares), the client (upstream),
// or the HAR collector (innermost — runs after httpretty has read the body).
func TestHARAndHTTPLoggingBodiesCoexist(t *testing.T) {
	g := gomega.NewWithT(t)

	const reqBody = `{"name":"alice","note":"hello"}`
	const respBody = `{"ok":true,"echo":"hello"}`

	var serverSawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		serverSawBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	defer server.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := testObservabilityContext{
		collector:  collector,
		harLevel:   logger.Trace,
		logHeaders: true,
		logBodies:  true,
	}
	client := &http.Client{Transport: applyHTTPObservability(ctx, "http", http.DefaultTransport, nil)}

	resp, err := client.Post(server.URL, "application/json", strings.NewReader(reqBody))
	g.Expect(err).ToNot(gomega.HaveOccurred())
	gotResp, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	g.Expect(serverSawBody).To(gomega.Equal(reqBody),
		"server must receive the full request body — proves httpretty restored it for the HAR layer below")
	g.Expect(string(gotResp)).To(gomega.Equal(respBody),
		"client must receive the full response body — proves both layers restored it on the way back")

	entries := collector.Entries()
	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Request.PostData).ToNot(gomega.BeNil(),
		"HAR captures request body — proves it was still readable after httpretty consumed it")
	// HAR may re-serialize JSON (key reordering, whitespace) so assert on content fragments.
	g.Expect(entries[0].Request.PostData.Text).To(gomega.ContainSubstring(`"name":"alice"`))
	g.Expect(entries[0].Request.PostData.Text).To(gomega.ContainSubstring(`"note":"hello"`))
	g.Expect(entries[0].Response.Content.Text).To(gomega.ContainSubstring(`"ok":true`),
		"HAR captures response body — proves it was still readable after httpretty consumed it")
	g.Expect(entries[0].Response.Content.Text).To(gomega.ContainSubstring(`"echo":"hello"`))
}
