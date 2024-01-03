package tests

import (
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/models"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Event queue views", func() {
	ginkgo.It("should query event queue views", func() {
		var summaries []models.EventQueueSummary
		err := DefaultContext.DB().Find(&summaries).Error
		Expect(err).ToNot(HaveOccurred())

		logger.Infof("eventQueueSummary (%d)", len(summaries))
	})

	ginkgo.It("should return deleted checks", func() {
		var summaries []models.PushQueueSummary
		err := DefaultContext.DB().Find(&summaries).Error
		Expect(err).ToNot(HaveOccurred())

		logger.Infof("pushQueueSummary (%d)", len(summaries))
	})
})
