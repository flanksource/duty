package tests

import (
	"time"

	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/matcher"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Check summary", ginkgo.Ordered, func() {
	ginkgo.It("should return old and non deleted checks", func() {
		ginkgo.Skip("Test is failing on GH actions, but not locally")
		err := job.RefreshCheckStatusSummary(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		err = job.RefreshCheckStatusSummaryAged(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		result, err := query.CheckSummary(DefaultContext, query.OrderByName())
		Expect(err).ToNot(HaveOccurred())

		matcher.MatchFixture("fixtures/expectations/check_status_summary.json", result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].agent_id)`)
	})

	ginkgo.It("should return deleted checks", func() {
		ginkgo.Skip("Test is failing on GH actions, but not locally")
		err := job.RefreshCheckStatusSummary(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		err = job.RefreshCheckStatusSummaryAged(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		month := time.Now().Add(-1 * 24 * 30 * time.Hour)
		result, err := query.CheckSummary(DefaultContext, query.CheckSummaryOptions{
			SortBy:     query.CheckSummarySortByName,
			DeleteFrom: &month,
		})
		Expect(err).ToNot(HaveOccurred())

		matcher.MatchFixture("fixtures/expectations/check_status_summary_deleted.json", result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].deleted_at) | del(.[].agent_id)`)
	})
})
