package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"time"

	"github.com/flanksource/commons/logger"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
	"github.com/flanksource/duty/logs/loki"
)

var _ = ginkgo.Describe("Loki Integration", ginkgo.Ordered, func() {
	var (
		lokiURL string
		ctx     context.Context
	)

	ginkgo.BeforeAll(func() {
		lokiURL = os.Getenv("LOKI_URL")
		if lokiURL == "" {
			lokiURL = "http://localhost:3100"
		}
		ctx = DefaultContext

		Eventually(func() error {
			resp, err := http.Get(lokiURL + "/ready")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return fmt.Errorf("loki not ready, status: %d", resp.StatusCode)
			}

			return nil
		}, 30*time.Second, 2*time.Second).Should(Succeed())

		err := injectLokiLogs(lokiURL, map[string]string{"source": "setup"})
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.Describe("Fetch", func() {
		ginkgo.It("should fetch logs successfully", func() {
			conn := connection.Loki{URL: lokiURL}
			lokiClient := loki.New(conn, nil)

			request := loki.Request{
				Query: `{job="test"}`,
				LogsRequestBase: logs.LogsRequestBase{
					Start: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					End:   time.Now().Format(time.RFC3339),
					Limit: "100",
				},
			}

			result, err := lokiClient.Search(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Logs).To(HaveLen(5))
		})

		ginkgo.It("should handle empty query results", func() {
			conn := connection.Loki{URL: lokiURL}
			lokiClient := loki.New(conn, nil)

			request := loki.Request{
				Query: `{job="nonexistent"}`,
				LogsRequestBase: logs.LogsRequestBase{
					Start: time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
					End:   time.Now().Format(time.RFC3339),
					Limit: "10",
				},
			}

			result, err := lokiClient.Search(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Logs).To(BeEmpty())
		})

		ginkgo.It("should handle invalid queries gracefully", func() {
			conn := connection.Loki{URL: lokiURL}
			lokiClient := loki.New(conn, nil)

			request := loki.Request{
				Query: `{invalid query syntax`,
				LogsRequestBase: logs.LogsRequestBase{
					Start: time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
					End:   time.Now().Format(time.RFC3339),
					Limit: "10",
				},
			}

			_, err := lokiClient.Search(ctx, request)
			Expect(err).To(HaveOccurred())
		})
	})

	ginkgo.Describe("Stream", func() {
		ginkgo.It("should establish streaming connection", func() {
			conn := connection.Loki{URL: lokiURL}
			lokiClient := loki.New(conn, nil)

			streamCtx, cancel := ctx.WithTimeout(5 * time.Second)
			defer cancel()

			request := loki.StreamRequest{
				Query:    `{job="test"}`,
				DelayFor: 0,
				Limit:    10,
			}

			logChan, err := lokiClient.Stream(streamCtx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(logChan).NotTo(BeNil())

			select {
			case item, ok := <-logChan:
				if !ok {
					logger.Infof("Log channel closed as expected")
				} else {
					Expect(item.Error).To(BeNil())
					logger.Infof("Received log line from stream as expected: %s", item.LogLine.Message)
					Expect(item.LogLine.Message).To(Equal("Test log message 1 - info level"))
				}
			case <-streamCtx.Done():
				logger.Infof("Stream context cancelled, log channel closed as expected")
			case <-time.After(10 * time.Second):
				ginkgo.Fail("Timed out waiting for log line from stream")
			}
		})

		ginkgo.It("should receive logs in streaming mode", func() {
			conn := connection.Loki{URL: lokiURL}
			lokiClient := loki.New(conn, nil)

			streamCtx, cancel := ctx.WithTimeout(30 * time.Second)
			defer cancel()

			// Start streaming with limit 0 to only get new logs
			request := loki.StreamRequest{
				Query:    `{job="test"}`,
				DelayFor: 0,
				Limit:    0,
			}

			logChan, err := lokiClient.Stream(streamCtx, request)
			Expect(err).NotTo(HaveOccurred())

			// Inject new log lines after stream is started
			go func() {
				time.Sleep(2 * time.Second) // Give stream time to be ready
				err := injectLokiLogs(lokiURL, map[string]string{"source": "streaming", "testcase": "live"})
				Expect(err).NotTo(HaveOccurred())
			}()

			// Wait for the newly injected logs
			var receivedLogs []*logs.LogLine
			timeout := time.After(15 * time.Second)

		receiveLogs:
			for {
				select {
				case item, ok := <-logChan:
					if !ok {
						break receiveLogs
					}
					Expect(item.Error).To(BeNil())
					receivedLogs = append(receivedLogs, item.LogLine)
					// Got at least one new log, test is successful
					if len(receivedLogs) >= 1 {
						break receiveLogs
					}

				case <-timeout:
					break receiveLogs
				}
			}

			// We should have received the newly injected logs
			Expect(len(receivedLogs)).To(BeNumerically(">=", 1))
		})
	})
})

type LogEntry struct {
	Timestamp int64
	Message   string
}

type LogStream struct {
	Labels  map[string]string
	Entries []LogEntry
}

func (ls LogStream) ToLokiFormat() map[string]any {
	values := make([][]string, len(ls.Entries))
	for i, entry := range ls.Entries {
		values[i] = []string{fmt.Sprintf("%d", entry.Timestamp), entry.Message}
	}

	return map[string]any{
		"stream": ls.Labels,
		"values": values,
	}
}

func injectLokiLogs(lokiURL string, extraLabels map[string]string) error {
	now := time.Now().UnixNano()

	streams := []LogStream{
		{
			Labels: map[string]string{
				"job":   "test",
				"level": "info",
				"host":  "test-host-1",
			},
			Entries: []LogEntry{
				{Timestamp: now, Message: "Test log message 1 - info level"},
				{Timestamp: now + 1000000, Message: "Test log message 2 - another info"},
				{Timestamp: now + 2000000, Message: "Test log message 3 - final info"},
			},
		},
		{
			Labels: map[string]string{
				"job":   "test",
				"level": "error",
				"host":  "test-host-2",
			},
			Entries: []LogEntry{
				{Timestamp: now + 3000000, Message: "Test error message 1"},
				{Timestamp: now + 4000000, Message: "Test error message 2"},
			},
		},
		{
			Labels: map[string]string{
				"job":      "production",
				"level":    "warn",
				"severity": "warning",
				"instance": "prod-server-1",
			},
			Entries: []LogEntry{
				{Timestamp: now + 5000000, Message: "Production warning message"},
				{Timestamp: now + 6000000, Message: "Another production warning"},
			},
		},
	}

	// Add extra labels to all streams
	for i := range streams {
		maps.Copy(streams[i].Labels, extraLabels)
	}

	// Convert streams to Loki format
	lokiStreams := make([]map[string]any, len(streams))
	for i, stream := range streams {
		lokiStreams[i] = stream.ToLokiFormat()
	}

	logData := map[string]any{
		"streams": lokiStreams,
	}

	jsonData, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("failed to marshal log data: %w", err)
	}

	resp, err := http.Post(
		lokiURL+"/loki/api/v1/push",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to push logs to loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to push logs, status code: %d", resp.StatusCode)
	}

	time.Sleep(2 * time.Second)
	return nil
}
