package tests

import (
	"fmt"
	"net/url"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/testutils"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("CheckDetails", ginkgo.Ordered, ginkgo.Focus, func() {
	ginkgo.It("should return old and non deleted checks", func() {
		urlParam := url.Values{
			"since": []string{"30d"},
			"check": []string{dummy.LogisticsAPIHealthHTTPCheck.ID.String()},
		}
		var q query.CheckQueryParams
		err := q.Init(urlParam)
		Expect(err).To(BeNil())

		ts, uptime, latency, err := q.ExecuteDetails(testutils.DefaultContext)
		Expect(err).To(BeNil())

		logger.Infof("ts: %d uptime: %v latency: %v", len(ts), uptime, latency)
		for _, t := range ts {
			fmt.Printf("timestamp: %s\n", t.Time)
		}
	})
})
