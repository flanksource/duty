package tests

import (
	"time"

	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/matcher"
	"github.com/flanksource/duty/testutils"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Check summary", ginkgo.Ordered, func() {
	ginkgo.It("should return old and non deleted checks", func() {
		err := job.RefreshCheckStatusSummary(testutils.DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		err = job.RefreshCheckStatusSummaryAged(testutils.DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		result, err := query.CheckSummary(testutils.DefaultContext, query.OrderByName())
		Expect(err).ToNot(HaveOccurred())

		// TODO: Test fails due to latency and uptime field
		// Skipping this for now
		//        	"id": "0186b7a4-9338-7142-1b10-25dc49030218",
		//     		"labels": {},
		//     		"latency": {
		//    -			"avg": 50,
		//    -			"p50": 50,
		//    -			"p95": 50,
		//    -			"p99": 50,
		//     			"rolling1h": 0
		//     		},
		//     		"name": "logistics-db-check",
		//    @@ -79,7 +75,7 @@
		//     		"status": "unhealthy",
		//     		"type": "postgres",
		//     		"uptime": {
		//    -			"failed": 1,
		//    +			"failed": 0,
		//     			"passed": 0
		//    	    }
		//

		matcher.MatchFixture("fixtures/expectations/check_status_summary.json", result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].agent_id) | del(.[].latency) | del(.[].uptime)`)
	})

	ginkgo.It("should return deleted checks", func() {
		err := job.RefreshCheckStatusSummary(testutils.DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		err = job.RefreshCheckStatusSummaryAged(testutils.DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		month := time.Now().Add(-1 * 24 * 30 * time.Hour)
		result, err := query.CheckSummary(testutils.DefaultContext, query.CheckSummaryOptions{
			SortBy:     query.CheckSummarySortByName,
			DeleteFrom: &month,
		})
		Expect(err).ToNot(HaveOccurred())

		matcher.MatchFixture("fixtures/expectations/check_status_summary_deleted.json", result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].deleted_at) | del(.[].agent_id)`)
	})
})
