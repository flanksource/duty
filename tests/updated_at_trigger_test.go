package tests

import (
	"time"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty"
	"github.com/flanksource/duty/models"
)

var _ = ginkgo.Describe("Test updated_at trigger behaviour", ginkgo.Ordered, func() {
	var (
		eightDaysAgo = time.Now().Add(-8 * 24 * time.Hour)

		config1 = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("pod-1"),
			Type:        lo.ToPtr("Kubernetes::Pod"),
			ConfigClass: "Pod",
		}
		config2 = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("deployment-1"),
			Type:        lo.ToPtr("Kubernetes::Deployment"),
			ConfigClass: "Deployment",
		}
		component1 = models.Component{
			ID:   uuid.New(),
			Name: "component-1",
			Type: "test",
		}
		component2 = models.Component{
			ID:   uuid.New(),
			Name: "component-2",
			Type: "test",
		}
	)

	ginkgo.BeforeAll(func() {
		// Set deleted_at to 8 days ago
		config1.DeletedAt = lo.ToPtr(eightDaysAgo)
		component1.DeletedAt = lo.ToPtr(eightDaysAgo)

		err := DefaultContext.DB().Create(&[]models.ConfigItem{config1, config2}).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Create(&[]models.Component{component1, component2}).Error
		Expect(err).To(BeNil())
	})

	ginkgo.It("should not reset deleted_at for already deleted items", func() {
		// Update deleted_at for both config items
		err := DefaultContext.DB().Model(&models.ConfigItem{}).
			Where("id IN (?)", uuid.UUIDs{config1.ID, config2.ID}).
			Update("deleted_at", duty.Now()).Error
		Expect(err).To(BeNil())

		c1, err := gorm.G[models.ConfigItem](DefaultContext.DB()).Where("id = ?", config1.ID).First(DefaultContext)
		Expect(err).To(BeNil())
		Expect(lo.FromPtr(c1.DeletedAt)).To(BeTemporally("~", eightDaysAgo, time.Minute), "deleted_at should not be updated")

		c2, err := gorm.G[models.ConfigItem](DefaultContext.DB()).Where("id = ?", config2.ID).First(DefaultContext)
		Expect(err).To(BeNil())
		Expect(lo.FromPtr(c2.DeletedAt)).ToNot(BeTemporally("~", eightDaysAgo, time.Minute), "deleted_at should be updated")

		// Update deleted_at for both components
		err = DefaultContext.DB().Model(&models.Component{}).
			Where("id IN (?)", uuid.UUIDs{component1.ID, component2.ID}).
			Update("deleted_at", duty.Now()).Error
		Expect(err).To(BeNil())

		comp1, err := gorm.G[models.Component](DefaultContext.DB()).Where("id = ?", component1.ID).First(DefaultContext)
		Expect(err).To(BeNil())
		Expect(lo.FromPtr(comp1.DeletedAt)).To(BeTemporally("~", eightDaysAgo, time.Minute), "deleted_at should not be updated")

		comp2, err := gorm.G[models.Component](DefaultContext.DB()).Where("id = ?", component2.ID).First(DefaultContext)
		Expect(err).To(BeNil())
		Expect(lo.FromPtr(comp2.DeletedAt)).ToNot(BeTemporally("~", eightDaysAgo, time.Minute), "deleted_at should be updated")
	})

	ginkgo.AfterAll(func() {
		// Cleanup
		DefaultContext.DB().Where("id IN (?)", uuid.UUIDs{config1.ID, config2.ID}).Delete(&models.ConfigItem{})
		DefaultContext.DB().Where("id IN (?)", uuid.UUIDs{component1.ID, component2.ID}).Delete(&models.Component{})
	})
})
