package tests

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("notification group resources", ginkgo.Ordered, func() {
	var group models.NotificationGroup
	var notification models.Notification

	ginkgo.BeforeAll(func() {
		notification = models.Notification{
			Name:     "group_resource_test",
			Events:   []string{"check.failed", "check.passed"},
			Source:   models.SourceCRD,
			Template: "test",
		}

		err := DefaultContext.DB().Create(&notification).Error
		Expect(err).To(BeNil())
	})

	ginkgo.AfterAll(func() {
		err := DefaultContext.DB().Delete(&group).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Where("id = ?", notification.ID).Delete(&notification).Error
		Expect(err).To(BeNil())
	})

	ginkgo.It("should upsert", func() {
		group = models.NotificationGroup{
			Hash:           "test",
			NotificationID: notification.ID,
		}

		err := DefaultContext.DB().Create(&group).Error
		Expect(err).To(BeNil())

		Expect(group.ID).ToNot(Equal(uuid.Nil))

		var updatedAt time.Time
		for range 5 {
			notificationGroupResource := models.NotificationGroupResource{
				GroupID:  group.ID,
				ConfigID: &dummy.KubernetesCluster.ID,
			}

			err = notificationGroupResource.Upsert(DefaultContext.DB())
			Expect(err).To(BeNil())

			var fetched models.NotificationGroupResource
			err = DefaultContext.DB().Where("group_id = ?", group.ID).Where("config_id = ?", dummy.KubernetesCluster.ID).First(&fetched).Error
			Expect(err).To(BeNil())

			Expect(*fetched.UpdatedAt).To(BeTemporally(">", updatedAt))
			updatedAt = *fetched.UpdatedAt
		}
	})
})

var _ = ginkgo.Describe("unsent notification", ginkgo.Ordered, func() {
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

	var _ = ginkgo.Describe("deduplication", func() {
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

	var _ = ginkgo.Describe("basic functionality", func() {

		ginkgo.It("should update body on duplicate notification", func() {
			var (
				dummyResource = uuid.New()
				sourceEvent   = notification.Events[0]
				sendStatus    = models.NotificationStatusSilenced
				firstBody     = "First notification body"
				updatedBody   = "Updated notification body"
			)

			// Insert first notification
			query := "SELECT * FROM insert_unsent_notification_to_history(?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, ?)"
			err := DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, sendStatus, silenceWindow, firstBody).Error
			Expect(err).To(BeNil())

			// Insert second notification with same parameters but different body
			query = "SELECT * FROM insert_unsent_notification_to_history(?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, ?)"
			err = DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, sendStatus, silenceWindow, updatedBody).Error
			Expect(err).To(BeNil())

			var sentHistories []models.NotificationSendHistory
			err = DefaultContext.DB().Model(&models.NotificationSendHistory{}).
				Where("status = ?", sendStatus).
				Where("resource_id = ?", dummyResource).
				Where("source_event = ?", sourceEvent).Find(&sentHistories).Error
			Expect(err).To(BeNil())
			Expect(len(sentHistories)).To(Equal(1), "Expected only one notification due to deduplication")

			sentHistory := sentHistories[0]
			Expect(sentHistory.Count).To(Equal(2), "Expected count to be 2 after duplicate insert")
			Expect(sentHistory.Body).ToNot(BeNil()) //nolint:staticcheck
			Expect(*sentHistory.Body).To(Equal(updatedBody), "Body should be updated to the newest value") //nolint:staticcheck
		})

		ginkgo.It("should save body_payload for unsent notifications", func() {
			var (
				dummyResource = uuid.New()
				sourceEvent   = notification.Events[0]
				sendStatus    = models.NotificationStatusSilenced
				bodyPayload   = `{"schema": "test", "data": {"key": "value"}}`
			)

			query := "SELECT * FROM insert_unsent_notification_to_history(?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, NULL, ?::jsonb)"
			err := DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, sendStatus, silenceWindow, bodyPayload).Error
			Expect(err).To(BeNil())

			var sentHistories []models.NotificationSendHistory
			err = DefaultContext.DB().Model(&models.NotificationSendHistory{}).
				Where("status = ?", sendStatus).
				Where("resource_id = ?", dummyResource).
				Where("source_event = ?", sourceEvent).Find(&sentHistories).Error
			Expect(err).To(BeNil())
			Expect(len(sentHistories)).To(Equal(1))

			sentHistory := sentHistories[0]
			Expect(sentHistory.ResourceID).To(Equal(dummyResource))
			Expect(sentHistory.Status).To(Equal(sendStatus))
			Expect(sentHistory.BodyPayload).ToNot(BeNil())
			Expect(string(sentHistory.BodyPayload)).To(MatchJSON(bodyPayload))
		})

		ginkgo.It("should update body_payload on duplicate notification", func() {
			var (
				dummyResource      = uuid.New()
				sourceEvent        = notification.Events[0]
				sendStatus         = models.NotificationStatusSilenced
				firstBodyPayload   = `{"schema": "test", "data": {"version": 1}}`
				updatedBodyPayload = `{"schema": "test", "data": {"version": 2}}`
			)

			// Insert first notification
			query := "SELECT * FROM insert_unsent_notification_to_history(?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, NULL, ?::jsonb)"
			err := DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, sendStatus, silenceWindow, firstBodyPayload).Error
			Expect(err).To(BeNil())

			// Insert second notification with same parameters but different body_payload
			err = DefaultContext.DB().Exec(query, notification.ID, sourceEvent, dummyResource, sendStatus, silenceWindow, updatedBodyPayload).Error
			Expect(err).To(BeNil())

			var sentHistories []models.NotificationSendHistory
			err = DefaultContext.DB().Model(&models.NotificationSendHistory{}).
				Where("status = ?", sendStatus).
				Where("resource_id = ?", dummyResource).
				Where("source_event = ?", sourceEvent).Find(&sentHistories).Error
			Expect(err).To(BeNil())
			Expect(len(sentHistories)).To(Equal(1), "Expected only one notification due to deduplication")

			sentHistory := sentHistories[0]
			Expect(sentHistory.Count).To(Equal(2), "Expected count to be 2 after duplicate insert")
			Expect(sentHistory.BodyPayload).ToNot(BeNil())
			Expect(string(sentHistory.BodyPayload)).To(MatchJSON(updatedBodyPayload), "BodyPayload should be updated to the newest value")
		})
	})
})
