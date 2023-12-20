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

		matcher.MatchFixture("fixtures/expectations/check_status_summary.json", result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].agent_id)`)
	})

	ginkgo.It("should return deleted checks", func() {
		err := job.RefreshCheckStatusSummary(testutils.DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		err = job.RefreshCheckStatusSummaryAged(testutils.DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		year := time.Now().Add(-1 * 24 * 365 * time.Hour)
		result, err := query.CheckSummary(testutils.DefaultContext, query.CheckSummaryOptions{
			SortBy:     query.CheckSummarySortByName,
			DeleteFrom: &year,
		})
		Expect(err).ToNot(HaveOccurred())

		matcher.MatchFixture("fixtures/expectations/check_status_summary_deleted.json", result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].deleted_at) | del(.[].agent_id)`)
	})
})
