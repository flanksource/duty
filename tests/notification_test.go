package tests

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("unsent notification deduplication", ginkgo.Ordered, func() {
	notification := models.Notification{
		Events: []string{"check.failed", "check.passed"},
		Source: models.SourceCRD,
	}

	var (
		dummyResources = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		statuses       = []string{models.NotificationStatusSilenced, models.NotificationStatusRepeatInterval}
		silenceWindow  = time.Second * 2
	)

	ginkgo.BeforeAll(func() {
		err := DefaultContext.DB().Create(&notification).Error
		Expect(err).To(BeNil())
	})

	for _, sourceEvent := range notification.Events {
		for _, sendStatus := range statuses {
			for i, dummyResource := range dummyResources {
				ginkgo.It(fmt.Sprintf("Event[%s] Resource[%d] should save unsent notifications to history", sourceEvent, i+1), func() {
					iteration := 10
					for j := 0; j < iteration; j++ {
						query := "SELECT * FROM insert_unsent_notification_to_history(?, ?, ?, ?, ?)"
						err := DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, sendStatus, silenceWindow).Error
						Expect(err).To(BeNil())
					}

					var sentHistories []models.NotificationSendHistory
					err := DefaultContext.DB().Model(&models.NotificationSendHistory{}).
						Where("status = ?", sendStatus).
						Where("resource_id = ?", dummyResource).
						Where("source_event = ?", sourceEvent).Find(&sentHistories).Error
					Expect(err).To(BeNil())
					Expect(len(sentHistories)).To(Equal(1))

					sentHistory := sentHistories[0]
					Expect(sentHistory.ResourceID).To(Equal(dummyResource))
					Expect(sentHistory.Status).To(Equal(sendStatus))
					Expect(sentHistory.Count).To(Equal(iteration))
					Expect(sentHistory.FirstObserved).To(BeTemporally("<", sentHistory.CreatedAt))
				})
			}
		}
	}

	ginkgo.It("should not dedup out of window", func() {
		time.Sleep(silenceWindow) // wait for window to pass

		var (
			dummyResource = dummyResources[0]
			sourceEvent   = notification.Events[0]
			sendStatus    = models.NotificationStatusSilenced
		)

		query := "SELECT * FROM insert_unsent_notification_to_history(?, ?, ?, ?, ?)"
		err := DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, models.NotificationStatusSilenced, silenceWindow).Error
		Expect(err).To(BeNil())

		var sentHistories []models.NotificationSendHistory
		err = DefaultContext.DB().Model(&models.NotificationSendHistory{}).
			Where("status = ?", sendStatus).
			Where("resource_id = ?", dummyResource).
			Where("source_event = ?", sourceEvent).Order("created_at DESC").Find(&sentHistories).Error
		Expect(err).To(BeNil())
		Expect(len(sentHistories)).To(Equal(2), "Expected 2 histories for two different window")

		sentHistory := sentHistories[0] // The first one is the latest

		Expect(sentHistory.ResourceID).To(Equal(dummyResource))
		Expect(sentHistory.Status).To(Equal(models.NotificationStatusSilenced))
		Expect(sentHistory.Count).To(Equal(1))
		Expect(sentHistory.FirstObserved.Unix()).To(Equal(sentHistory.CreatedAt.Unix()))
	})
})
