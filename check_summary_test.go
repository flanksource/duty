package duty

import (
	"encoding/json"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testCheckSummaryJSON(path string) {
	result, err := QueryCheckSummary(testDBPGPool)
	Expect(err).ToNot(HaveOccurred())

	resultJSON, err := json.Marshal(result)
	Expect(err).ToNot(HaveOccurred())

	expected := readTestFile(path)
	jqExpr := `del(.[].uptime.last_pass) | del(.[].uptime.last_fail)`
	matchJSON([]byte(expected), resultJSON, &jqExpr)
}

var _ = ginkgo.Describe("Check summary behavior", ginkgo.Ordered, func() {
	ginkgo.It("Should test check summary result", func() {
		err := RefreshCheckStatusSummary(testDBPGPool)
		Expect(err).ToNot(HaveOccurred())

		testCheckSummaryJSON("fixtures/expectations/check_status_summary.json")
	})
})
