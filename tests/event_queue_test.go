package tests

import (
	"fmt"

	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/postq"
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
			uuuid, err := hash.DeterministicUUID(fmt.Sprintf("%s-%d", eventName, i))
			Expect(err).ToNot(HaveOccurred())

			DefaultContext.DB().Create(&models.Event{
				Name:    eventName,
				EventID: uuuid,
				Properties: map[string]string{
					"id": fmt.Sprintf("%d", i),
				},
			})
		}

		consumer.ConsumeUntilEmpty(DefaultContext)
		Expect(handlerInvocationCount).To(Equal(iterations))
	})

	ginkgo.DescribeTable("should enforce uniqueness constraint on name and event_id",
		func(events []models.Event, shouldFail bool) {
			for i, event := range events {
				err := DefaultContext.DB().Create(&event).Error
				if i == 0 {
					// First event should always succeed
					Expect(err).To(BeNil())
				} else {
					if shouldFail {
						Expect(err).To(HaveOccurred(), "Expected duplicate event to fail unique constraint")
					} else {
						Expect(err).ToNot(HaveOccurred(), "Expected event creation to succeed")
					}
				}
			}
		},
		ginkgo.Entry("duplicate events with same name and properties id should fail",
			[]models.Event{
				{
					Name:       "test.unique.duplicate",
					EventID:    uuid.MustParse("4c185f4e-aaf8-4039-ae2f-84bbb7094f66"),
					Properties: map[string]string{"id": "12345"},
				},
				{
					Name:       "test.unique.duplicate",
					EventID:    uuid.MustParse("4c185f4e-aaf8-4039-ae2f-84bbb7094f66"),
					Properties: map[string]string{"id": "123"},
				},
			},
			true,
		),
		ginkgo.Entry("events with same name but different properties id should succeed",
			[]models.Event{
				{
					Name:       "test.unique.different-id",
					EventID:    uuid.New(),
					Properties: map[string]string{"id": "456"},
				},
				{
					Name:       "test.unique.different-id",
					EventID:    uuid.New(),
					Properties: map[string]string{"id": "456"},
				},
			},
			false,
		),
		ginkgo.Entry("events with different names but same properties id should succeed",
			[]models.Event{
				{
					Name:       "test.unique.different-name-1",
					EventID:    uuid.MustParse("9c82da24-4896-4341-9df2-07d7a4437a94"),
					Properties: map[string]string{"id": "999"},
				},
				{
					Name:       "test.unique.different-name-2",
					EventID:    uuid.MustParse("9c82da24-4896-4341-9df2-07d7a4437a94"),
					Properties: map[string]string{"id": "999"},
				},
			},
			false,
		),
	)

	ginkgo.It("should handle OnConflict using EventQueueUniqueConstraint", func() {
		event := models.Event{
			Name:       "test.unique.constraint",
			Properties: map[string]string{"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
		}

		// Create the first event
		err := DefaultContext.DB().Create(&event).Error
		Expect(err).ToNot(HaveOccurred())

		// Try to create the same event again with OnConflict DoNothing
		duplicateEvent := models.Event{
			Name:    "test.unique.constraint",
			EventID: event.EventID,
		}

		err = DefaultContext.DB().Clauses(clause.OnConflict{
			Columns:   models.EventQueueUniqueConstraint(),
			DoNothing: true,
		}).Create(&duplicateEvent).Error
		Expect(err).ToNot(HaveOccurred(), "OnConflict DoNothing should handle duplicate gracefully")

		// Count the number of events in the event queue
		var count int64
		err = DefaultContext.DB().Model(&models.Event{}).Where("name = ?", "test.unique.constraint").Count(&count).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(int64(1)))
	})
})
