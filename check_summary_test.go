package duty

import (
	"encoding/json"

	"github.com/flanksource/commons/logger"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testCheckSummaryJSON(path string) {
	result, err := QueryCheckSummary(testDBPGPool)
	Expect(err).ToNot(HaveOccurred())

	resultJSON, err := json.Marshal(result)
	logger.Infof("RESULT %s", string(resultJSON))
	Expect(err).ToNot(HaveOccurred())

	expected := readTestFile(path)
	jqExpr := `del(.[].uptime.last_pass) | del(.[].uptime.last_fail)`
	matchJSON([]byte(expected), resultJSON, &jqExpr)
}

var _ = ginkgo.Describe("Check summary behavior", ginkgo.Ordered, func() {
	ginkgo.FIt("Should test check summary result", func() {
		RefreshCheckStatusSummary(testDBPGPool)
		testCheckSummaryJSON("fixtures/expectations/check_status_summary.json")
	})
})
