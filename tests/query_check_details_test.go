package tests

import (
	"fmt"
	"net/url"
	"time"

	"github.com/flanksource/commons/duration"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("CheckDetails", ginkgo.Ordered, ginkgo.Focus, func() {
	type testRecord struct {
		since    string
		statuses int
		passed   int
		failed   int
		latency  types.Latency
	}

	testData := []testRecord{
		{since: "1w", statuses: 2, passed: 56, failed: 14, latency: types.Latency{Percentile99: 560, Percentile97: 0, Percentile95: 0}},
		{since: "1d", statuses: 5, passed: 56, failed: 14, latency: types.Latency{Percentile99: 1150, Percentile97: 570, Percentile95: 70}},
		{since: "1h", statuses: 61, passed: 48, failed: 13, latency: types.Latency{Percentile99: 1210, Percentile97: 1170, Percentile95: 1130}},
		{since: "30m", statuses: 31, passed: 24, failed: 7, latency: types.Latency{Percentile99: 610, Percentile97: 570, Percentile95: 530}},
	}

	for i := range testData {
		td := testData[i]

		ginkgo.It(fmt.Sprintf("since: %s", td.since), func() {
			parsed, err := duration.ParseDuration(td.since)
			Expect(err).To(BeNil())

			urlParam := url.Values{
				"since": []string{dummy.CurrentTime.Add(-time.Duration(parsed)).Format(time.RFC3339)},
				"end":   []string{dummy.CurrentTime.Format(time.RFC3339)},
				"check": []string{dummy.LogisticsAPIHealthHTTPCheck.ID.String()},
			}

			var q query.CheckQueryParams
			err = q.Init(urlParam)
			Expect(err).To(BeNil())

			ts, uptime, latency, err := q.ExecuteDetails(DefaultContext)
			Expect(err).To(BeNil())

			Expect(len(ts)).To(Equal(td.statuses), "unexpected number of results")
			Expect(uptime.Passed).To(Equal(td.passed), "unexpected passed checks")
			Expect(uptime.Failed).To(Equal(td.failed), "unexpected failed checks")
			Expect(latency).To(Equal(td.latency), "unexpected latency")
		})
	}
})
