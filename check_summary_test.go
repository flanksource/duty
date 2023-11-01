package duty

import (
	"context"

	"github.com/flanksource/duty/testutils"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testCheckSummaryJSON(path string) {
	result, err := QueryCheckSummary(context.Background(), testutils.TestDBPGPool, OrderByName())
	Expect(err).ToNot(HaveOccurred())

	match(path, result, `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at) | del(.[].agent_id)`)
}

var _ = ginkgo.Describe("Check summary behavior", ginkgo.Ordered, func() {
	ginkgo.It("Should test check summary result", func() {
		err := RefreshCheckStatusSummary(testutils.TestDBPGPool)
		Expect(err).ToNot(HaveOccurred())

		testCheckSummaryJSON("fixtures/expectations/check_status_summary.json")
	})
})
