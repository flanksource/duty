package tests

import (
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
)

var _ = ginkgo.Describe("Config Health Triggers", ginkgo.Ordered, func() {
	var ci *models.ConfigItem

	ginkgo.BeforeAll(func() {
		ci = &models.ConfigItem{
			ID:     uuid.New(),
			Type:   lo.ToPtr("Kubernetes::Pod"),
			Name:   lo.ToPtr("dummy-config-for-trigger"),
			Health: lo.ToPtr(models.HealthHealthy),
		}
		err := DefaultContext.DB().Create(ci).Error
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		DefaultContext.DB().Where("properties->>'id' = ?", ci.ID.String()).Delete(&models.Event{})
		DefaultContext.DB().Delete(ci)
	})

	ginkgo.BeforeEach(func() {
		DefaultContext.DB().Where("properties->>'id' = ?", ci.ID.String()).Delete(&models.Event{})
	})

	ginkgo.It("should NOT trigger events when inserting as healthy", func() {
		var events []models.Event
		err := DefaultContext.DB().Where("properties->>'id' = ?", ci.ID.String()).Find(&events).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(events).To(BeEmpty())
	})

	ginkgo.It("should emit config.degraded when transitioning from unhealthy to warning", func() {
		ci.Health = lo.ToPtr(models.HealthUnhealthy)
		err := DefaultContext.DB().Save(ci).Error
		Expect(err).ToNot(HaveOccurred())

		ci.Health = lo.ToPtr(models.HealthWarning)
		err = DefaultContext.DB().Save(ci).Error
		Expect(err).ToNot(HaveOccurred())

		var events []models.Event
		err = DefaultContext.DB().Where("properties->>'id' = ? AND name = ?", ci.ID.String(), "config.degraded").Find(&events).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(events).To(HaveLen(1))
	})

	ginkgo.It("should emit config.warning when transitioning from healthy to warning", func() {
		ci.Health = lo.ToPtr(models.HealthHealthy)
		err := DefaultContext.DB().Save(ci).Error
		Expect(err).ToNot(HaveOccurred())

		ci.Health = lo.ToPtr(models.HealthWarning)
		err = DefaultContext.DB().Save(ci).Error
		Expect(err).ToNot(HaveOccurred())

		var events []models.Event
		err = DefaultContext.DB().Where("properties->>'id' = ? AND name = ?", ci.ID.String(), "config.warning").Find(&events).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(events).To(HaveLen(1))
	})
})
