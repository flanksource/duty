package duty

import (
	"encoding/json"

	"github.com/flanksource/duty/hack"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testCheckSummaryJSON(path string) {
	result, err := QueryCheckSummary(hack.TestDBPGPool)
	Expect(err).ToNot(HaveOccurred())

	resultJSON, err := json.Marshal(result)
	Expect(err).ToNot(HaveOccurred())

	expected := readTestFile(path)
	jqExpr := `del(.[].uptime.last_pass) | del(.[].uptime.last_fail) | del(.[].created_at) | del(.[].updated_at)`
	matchJSON([]byte(expected), resultJSON, &jqExpr)
}

var _ = ginkgo.Describe("Check summary behavior", ginkgo.Ordered, func() {
	ginkgo.It("Should test check summary result", func() {
		err := RefreshCheckStatusSummary(hack.TestDBPGPool)
		Expect(err).ToNot(HaveOccurred())

		testCheckSummaryJSON("fixtures/expectations/check_status_summary.json")
	})
})
