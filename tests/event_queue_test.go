package tests

import (
	"time"

	"github.com/flanksource/commons/logger"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type eventQueueSummary struct {
	Name          string     `json:"name"`
	Pending       int64      `json:"pending"`
	Failed        int64      `json:"failed"`
	AvgAttempts   int64      `json:"average_attempts"`
	FirstFailure  *time.Time `json:"first_failure,omitempty"`
	LastFailure   *time.Time `json:"last_failure,omitempty"`
	MostCommonErr string     `json:"most_common_error,omitempty"`
}

func (t *eventQueueSummary) TableName() string {
	return "event_queue_summary"
}

type pushQueueSummary struct {
	Table         string     `json:"table"`
	Pending       int64      `json:"pending"`
	Failed        int64      `json:"failed"`
	AvgAttempts   int64      `json:"average_attempts"`
	FirstFailure  *time.Time `json:"first_failure,omitempty"`
	LastFailure   *time.Time `json:"last_failure,omitempty"`
	MostCommonErr string     `json:"most_common_error,omitempty"`
}

func (t *pushQueueSummary) TableName() string {
	return "push_queue_summary"
}

var _ = ginkgo.Describe("Event queue views", func() {
	ginkgo.It("should query event queue views", func() {
		var summaries []eventQueueSummary
		err := DefaultContext.DB().Find(&summaries).Error
		Expect(err).ToNot(HaveOccurred())

		logger.Infof("eventQueueSummary (%d)", len(summaries))
	})

	ginkgo.It("should return deleted checks", func() {
		var summaries []pushQueueSummary
		err := DefaultContext.DB().Find(&summaries).Error
		Expect(err).ToNot(HaveOccurred())

		logger.Infof("pushQueueSummary (%d)", len(summaries))
	})
})
