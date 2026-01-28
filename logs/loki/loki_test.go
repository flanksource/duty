package loki_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/flanksource/commons/logger"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	dutyCtx "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
	"github.com/flanksource/duty/logs/loki"
)

func TestLoki(t *testing.T) {
	logger.Use(ginkgo.GinkgoWriter)
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Loki Suite")
}

var _ = ginkgo.Describe("Loki Integration", ginkgo.Ordered, func() {
	var (
		lokiServer *loki.Server
		ctx        dutyCtx.Context
		tempDir    string
	)

	ginkgo.BeforeAll(func() {
		ctx = dutyCtx.NewContext(context.Background())
		tempDir, _ = os.MkdirTemp("", "loki")
		lokiServer = loki.NewServer(loki.ServerConfig{DataPath: tempDir})
		err := lokiServer.Start()
		Expect(err).NotTo(HaveOccurred())

		err = lokiServer.UploadLogs(testLogStreams(), map[string]string{"source": "setup"})
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if lokiServer != nil {
			_ = lokiServer.Stop()
		}
	})

	ginkgo.Describe("Fetch", func() {
		ginkgo.It("should fetch logs successfully", func() {
			conn := connection.Loki{URL: lokiServer.URL()}
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
			conn := connection.Loki{URL: lokiServer.URL()}
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
			conn := connection.Loki{URL: lokiServer.URL()}
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
			conn := connection.Loki{URL: lokiServer.URL()}
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
			conn := connection.Loki{URL: lokiServer.URL()}
			lokiClient := loki.New(conn, nil)

			streamCtx, cancel := ctx.WithTimeout(30 * time.Second)
			defer cancel()

			request := loki.StreamRequest{
				Query:    `{job="test"}`,
				DelayFor: 0,
				Limit:    0,
			}

			logChan, err := lokiClient.Stream(streamCtx, request)
			Expect(err).NotTo(HaveOccurred())

			injectErr := make(chan error, 1)
			go func() {
				time.Sleep(2 * time.Second)
				injectErr <- lokiServer.UploadLogs(testLogStreams(), map[string]string{"source": "streaming", "testcase": "live"})
			}()

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
					if len(receivedLogs) >= 1 {
						break receiveLogs
					}

				case err := <-injectErr:
					Expect(err).NotTo(HaveOccurred())

				case <-timeout:
					break receiveLogs
				}
			}

			Expect(len(receivedLogs)).To(BeNumerically(">=", 1))
		})
	})
})

func testLogStreams() []loki.LogStream {
	now := time.Now().UnixNano()
	return []loki.LogStream{
		{
			Labels: map[string]string{
				"job":   "test",
				"level": "info",
				"host":  "test-host-1",
			},
			Entries: []loki.LogEntry{
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
			Entries: []loki.LogEntry{
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
			Entries: []loki.LogEntry{
				{Timestamp: now + 5000000, Message: "Production warning message"},
				{Timestamp: now + 6000000, Message: "Another production warning"},
			},
		},
	}
}
