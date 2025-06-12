package e2e

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/logs/gcpcloudlogging"
)

// Test to run manually for debugging purpose
// TODO: remove this
var _ = ginkgo.Describe("TestGCPCloudLoggingStreams", ginkgo.Pending, func() {
	ginkgo.It("should stream logs", func() {
		conn := connection.GCPConnection{
			Project:        "workload-prod-eu-02",
			ConnectionName: "connection://mc/gcloud-flanksource",
		}
		gcp, err := gcpcloudlogging.New(DefaultContext, conn, nil)
		Expect(err).NotTo(HaveOccurred())

		// {
		// 	response, err := gcp.Search(DefaultContext, gcpcloudlogging.Request{
		// 		LogsRequestBase: logs.LogsRequestBase{
		// 			Limit: "5",
		// 			Start: "now",
		// 		},
		// 	})
		// 	Expect(err).NotTo(HaveOccurred())

		// 	fmt.Println(response.Metadata)
		// return
		// }

		// Create stream request for recent logs
		req := gcpcloudlogging.StreamRequest{
			Start: "now-1h", // Last hour of logs
		}

		var logChan <-chan gcpcloudlogging.StreamItem

		// Start streaming in a goroutine
		go func() {
			logChan, err = gcp.Stream(DefaultContext, req)
			Expect(err).To(BeNil())
		}()

		// Collect logs for a short duration
		timeout := time.After(10 * time.Second)
		logCount := 0

		for {
			select {
			case logLine, ok := <-logChan:
				if !ok {
					return
				}

				fmt.Println(logLine)
				logCount++

				// Stop after getting some logs
				if logCount >= 5 {
					return
				}

			case <-timeout:
				ginkgo.Fail("did not stream any logs")
				return
			}
		}
	})
})
