package tests

import (
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Config detail", func() {
	ginkgo.It("reports only the config's own changes", func() {
		Expect(job.RefreshConfigItemSummary7d(DefaultContext)).To(Succeed())

		var expectedChanges int
		err := DefaultContext.DB().Raw(
			"SELECT config_changes_count FROM config_item_summary_7d WHERE config_id = ?",
			dummy.NginxIngressPod.ID,
		).Scan(&expectedChanges).Error
		Expect(err).NotTo(HaveOccurred())

		var changes int
		err = DefaultContext.DB().Raw(
			"SELECT (summary->>'changes')::int FROM config_detail WHERE id = ?",
			dummy.NginxIngressPod.ID,
		).Scan(&changes).Error
		Expect(err).NotTo(HaveOccurred())
		Expect(changes).To(Equal(expectedChanges))
	})
})
