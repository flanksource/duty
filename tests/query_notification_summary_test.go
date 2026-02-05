package tests

import (
	"encoding/json"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

type NotificationSummaryGroupByResource struct {
	LastSeen                  time.Time
	FirstObserved             time.Time
	Error                     int
	Sent                      int
	Suppressed                int
	Resource                  map[string]any
	ResourceHealth            models.Health            `json:"resource_health"`
	ResourceStatus            models.CheckHealthStatus `json:"resource_status"`
	ResourceHealthDescription string                   `json:"resource_health_description"`
}

var _ = ginkgo.Describe("Notification Summary", ginkgo.Ordered, ginkgo.Serial, func() {
	ginkgo.BeforeAll(func() {
		referenceTime := time.Now()
		sendHistories := []models.NotificationSendHistory{
			{
				NotificationID:            dummy.NoMatchNotification.ID,
				ResourceHealth:            models.HealthUnhealthy,
				ResourceStatus:            "ImagePullBackOff",
				ResourceHealthDescription: "image not found",
				ResourceID:                dummy.LogisticsAPIPodConfig.ID,
				CreatedAt:                 referenceTime.Add(-time.Minute * 20),
				FirstObserved:             referenceTime.Add(-time.Minute * 20),
				Status:                    models.NotificationStatusSent,
				SourceEvent:               "config.unhealthy",
			},
			{
				NotificationID:            dummy.NoMatchNotification.ID,
				ResourceHealth:            models.HealthHealthy,
				ResourceStatus:            "Running",
				ResourceHealthDescription: "",
				ResourceID:                dummy.LogisticsAPIPodConfig.ID,
				CreatedAt:                 referenceTime.Add(-time.Minute * 15),
				FirstObserved:             referenceTime.Add(-time.Minute * 15),
				Status:                    models.NotificationStatusSent,
				SourceEvent:               "config.healthy",
			},
			{
				NotificationID:            dummy.NoMatchNotification.ID,
				ResourceHealth:            models.HealthUnhealthy,
				ResourceStatus:            "CrashLoopBackOff",
				ResourceHealthDescription: "application failed",
				ResourceID:                dummy.LogisticsAPIPodConfig.ID,
				CreatedAt:                 referenceTime.Add(-time.Minute * 10),
				FirstObserved:             referenceTime.Add(-time.Minute * 10),
				Status:                    models.NotificationStatusSent,
				SourceEvent:               "config.unhealthy",
			},
			{
				NotificationID:            dummy.NoMatchNotification.ID,
				ResourceHealth:            models.HealthHealthy,
				ResourceStatus:            "Running",
				ResourceHealthDescription: "",
				ResourceID:                dummy.LogisticsAPIPodConfig.ID,
				CreatedAt:                 referenceTime.Add(-time.Minute * 5),
				FirstObserved:             referenceTime.Add(-time.Minute * 5),
				Status:                    models.NotificationStatusSent,
				SourceEvent:               "config.healthy",
			},
		}
		Expect(DefaultContext.DB().Create(&sendHistories).Error).ToNot(HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		Expect(DefaultContext.DB().Where("notification_id = ?", dummy.NoMatchNotification.ID).Delete(&models.NotificationSendHistory{}).Error).ToNot(HaveOccurred())
	})

	ginkgo.It("should return the correct notification summary", func() {
		request := query.NotificationSendHistorySummaryRequest{
			Search:       *dummy.LogisticsAPIPodConfig.Name,
			ResourceKind: query.NotificationResourceKindConfig,
			From:         "now-18m",
		}
		notificationSummary, err := query.NotificationSendHistorySummary(DefaultContext, request)
		Expect(err).ToNot(HaveOccurred())

		Expect(notificationSummary.Total).To(Equal(int64(1)), "record for only the given resource")

		var result []NotificationSummaryGroupByResource
		err = json.Unmarshal(notificationSummary.Results, &result)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(result)).To(Equal(1))
		Expect(result[0].Sent).To(Equal(3), "only need 3 sent as the first one falls out of range")
	})
})
