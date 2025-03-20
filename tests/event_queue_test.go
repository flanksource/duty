package tests

import (
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/postq"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Event queue", func() {
	ginkgo.It("should query event queue views", func() {
		var summaries []models.EventQueueSummary
		err := DefaultContext.DB().Find(&summaries).Error
		Expect(err).ToNot(HaveOccurred())

		logger.Infof("eventQueueSummary (%d)", len(summaries))
	})

	ginkgo.It("should process event queue one at a time", func() {
		const iterations = 10
		const eventName = "test.one-at-a-time"
		var handlerInvocationCount int // number of times the event handler is called

		syncConsumer := postq.SyncEventConsumer{
			WatchEvents: []string{eventName},
			Consumers: []postq.SyncEventHandlerFunc{
				func(ctx context.Context, e models.Event) error {
					handlerInvocationCount++
					return nil
				},
			},
		}

		consumer, err := syncConsumer.EventConsumer()
		Expect(err).ToNot(HaveOccurred())

		for i := range iterations {
			DefaultContext.DB().Create(&models.Event{
				Name: eventName,
				Properties: map[string]string{
					"id": fmt.Sprintf("%d", i),
				},
			})
		}

		consumer.ConsumeUntilEmpty(DefaultContext)
		Expect(handlerInvocationCount).To(Equal(iterations))
	})
})
