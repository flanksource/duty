package tests

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var _ = ginkgo.Describe("Delete old config items", ginkgo.Ordered, func() {
	var (
		// Config items that should be deleted (older than 7 days)
		oldConfig1 = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("old-pod-1"),
			Type:        lo.ToPtr("Kubernetes::Pod"),
			ConfigClass: "Pod",
			Tags:        types.JSONStringMap{"test": "delete-old-config-items"},
		}
		oldConfig2 = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("old-deployment-1"),
			Type:        lo.ToPtr("Kubernetes::Deployment"),
			ConfigClass: "Deployment",
			Tags:        types.JSONStringMap{"test": "delete-old-config-items"},
		}
		oldConfig3 = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("old-service-1"),
			Type:        lo.ToPtr("Kubernetes::Service"),
			ConfigClass: "Service",
			ParentID:    lo.ToPtr(oldConfig2.ID),
			Tags:        types.JSONStringMap{"test": "delete-old-config-items"},
		}

		// Config items that should NOT be deleted (recent or protected)
		recentConfig = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("recent-pod"),
			Type:        lo.ToPtr("Kubernetes::Pod"),
			ConfigClass: "Pod",
			Tags:        types.JSONStringMap{"test": "delete-old-config-items"},
		}
		protectedConfig = models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("protected-pod"),
			Type:        lo.ToPtr("Kubernetes::Pod"),
			ConfigClass: "Pod",
			Tags:        types.JSONStringMap{"test": "delete-old-config-items"},
		}

		// Related records for old configs
		oldConfigAnalysis1 = models.ConfigAnalysis{
			ID:       uuid.New(),
			ConfigID: oldConfig1.ID,
			Analyzer: "test-analyzer",
			Severity: models.SeverityInfo,
		}
		oldConfigChange1 = models.ConfigChange{
			ID:         uuid.New().String(),
			ConfigID:   oldConfig1.ID.String(),
			ChangeType: "test-change",
		}
		oldConfigRelationship1 = models.ConfigRelationship{
			ConfigID:  oldConfig1.ID.String(),
			RelatedID: oldConfig2.ID.String(),
			Relation:  "test-relation",
		}

		// Component that references protectedConfig (should prevent deletion)
		protectedComponent = models.Component{
			ID:       uuid.New(),
			Name:     "protected-component",
			Type:     "test",
			ConfigID: &protectedConfig.ID,
		}

		// Batch testing: Create multiple old configs to test batching (1500 items)
		batchOldConfigs        []models.ConfigItem
		batchOldConfigAnalysis []models.ConfigAnalysis
	)

	ginkgo.BeforeAll(func() {
		// Set deleted_at for old configs to 8 days ago (older than 7 days)
		eightDaysAgo := time.Now().Add(-8 * 24 * time.Hour)
		oldConfig1.DeletedAt = lo.ToPtr(eightDaysAgo)
		oldConfig2.DeletedAt = lo.ToPtr(eightDaysAgo)
		oldConfig3.DeletedAt = lo.ToPtr(eightDaysAgo)
		protectedConfig.DeletedAt = lo.ToPtr(eightDaysAgo)

		// Set deleted_at for recent config to 6 days ago (should not be deleted)
		sixDaysAgo := time.Now().Add(-6 * 24 * time.Hour)
		recentConfig.DeletedAt = lo.ToPtr(sixDaysAgo)

		// Create 1500 old config items for batch testing
		for i := 0; i < 1500; i++ {
			configID := uuid.New()
			batchOldConfigs = append(batchOldConfigs, models.ConfigItem{
				ID:          configID,
				Name:        lo.ToPtr(fmt.Sprintf("batch-pod-%d", i)),
				Type:        lo.ToPtr("Kubernetes::Pod"),
				ConfigClass: "Pod",
				Tags:        types.JSONStringMap{"test": "delete-old-config-items-batch"},
				DeletedAt:   lo.ToPtr(eightDaysAgo),
			})

			// Add some related records to test cascading deletes
			if i%10 == 0 {
				batchOldConfigAnalysis = append(batchOldConfigAnalysis, models.ConfigAnalysis{
					ID:       uuid.New(),
					ConfigID: configID,
					Analyzer: "batch-analyzer",
					Severity: models.SeverityInfo,
				})
			}
		}

		// Insert all config items
		err := DefaultContext.DB().Create(&[]models.ConfigItem{
			oldConfig1, oldConfig2, oldConfig3, recentConfig, protectedConfig,
		}).Error
		Expect(err).To(BeNil())

		// Insert batch config items in chunks to avoid memory issues
		chunkSize := 500
		for i := 0; i < len(batchOldConfigs); i += chunkSize {
			end := i + chunkSize
			if end > len(batchOldConfigs) {
				end = len(batchOldConfigs)
			}
			chunk := batchOldConfigs[i:end]
			err := DefaultContext.DB().Create(&chunk).Error
			Expect(err).To(BeNil())
		}

		// Insert related records
		err = DefaultContext.DB().Create(&oldConfigAnalysis1).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Create(&oldConfigChange1).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Create(&oldConfigRelationship1).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Create(&protectedComponent).Error
		Expect(err).To(BeNil())

		// Insert batch config analysis
		for i := 0; i < len(batchOldConfigAnalysis); i += chunkSize {
			end := i + chunkSize
			if end > len(batchOldConfigAnalysis) {
				end = len(batchOldConfigAnalysis)
			}
			chunk := batchOldConfigAnalysis[i:end]
			err := DefaultContext.DB().Create(&chunk).Error
			Expect(err).To(BeNil())
		}
	})

	ginkgo.It("should delete old config items and related records, handle batch deletions correctly", func() {
		// Count how many batch configs exist before deletion
		var countBefore int64
		err := DefaultContext.DB().Model(&models.ConfigItem{}).
			Where("tags->>'test' = 'delete-old-config-items-batch'").Count(&countBefore).Error
		Expect(err).To(BeNil())
		Expect(countBefore).To(Equal(int64(1500)), "should have 1500 batch configs before deletion")

		// Call the procedure to delete config items older than 7 days
		// This should process them in batches of 1000, committing after each batch
		err = DefaultContext.DB().Exec("CALL delete_old_config_items(7)").Error
		Expect(err).To(BeNil())

		// Verify old configs are deleted (hard delete, so they won't exist even with Unscoped)
		var count int64
		err = DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", oldConfig1.ID).Count(&count).Error
		Expect(err).To(BeNil())
		Expect(count).To(Equal(int64(0)), "oldConfig1 should be deleted")

		err = DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", oldConfig2.ID).Count(&count).Error
		Expect(err).To(BeNil())
		Expect(count).To(Equal(int64(0)), "oldConfig2 should be deleted")

		err = DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", oldConfig3.ID).Count(&count).Error
		Expect(err).To(BeNil())
		Expect(count).To(Equal(int64(0)), "oldConfig3 should be deleted")

		// Verify related records are deleted
		err = DefaultContext.DB().Model(&models.ConfigAnalysis{}).Where("id = ?", oldConfigAnalysis1.ID).Count(&count).Error
		Expect(err).To(BeNil())
		Expect(count).To(Equal(int64(0)))

		err = DefaultContext.DB().Model(&models.ConfigChange{}).Where("id = ?", oldConfigChange1.ID).Count(&count).Error
		Expect(err).To(BeNil())
		Expect(count).To(Equal(int64(0)))

		err = DefaultContext.DB().Model(&models.ConfigRelationship{}).
			Where("config_id = ? OR related_id = ?", oldConfig1.ID.String(), oldConfig1.ID.String()).Count(&count).Error
		Expect(err).To(BeNil())
		Expect(count).To(Equal(int64(0)))

		// Verify recent config is NOT deleted
		var foundRecentConfig models.ConfigItem
		err = DefaultContext.DB().Where("id = ?", recentConfig.ID).First(&foundRecentConfig).Error
		Expect(err).To(BeNil())
		Expect(foundRecentConfig.ID).To(Equal(recentConfig.ID))

		// Verify protected config is NOT deleted (has component reference)
		var foundProtectedConfig models.ConfigItem
		err = DefaultContext.DB().Where("id = ?", protectedConfig.ID).First(&foundProtectedConfig).Error
		Expect(err).To(BeNil(), "protectedConfig should still exist")
		Expect(foundProtectedConfig.ID).To(Equal(protectedConfig.ID))

		// Verify component's config_id still points to the protected config (unchanged)
		var foundComponent models.Component
		err = DefaultContext.DB().Where("id = ?", protectedComponent.ID).First(&foundComponent).Error
		Expect(err).To(BeNil())
		Expect(foundComponent.ConfigID).NotTo(BeNil(), "component config_id should not be null")
		Expect(*foundComponent.ConfigID).To(Equal(protectedConfig.ID), "component should still reference protectedConfig")

		// Verify all batch configs are deleted
		var countAfter int64
		err = DefaultContext.DB().Model(&models.ConfigItem{}).
			Where("tags->>'test' = 'delete-old-config-items-batch'").Count(&countAfter).Error
		Expect(err).To(BeNil())
		Expect(countAfter).To(Equal(int64(0)), "all batch configs should be deleted")

		// Verify related config analysis records are also deleted
		var analysisCount int64
		err = DefaultContext.DB().Model(&models.ConfigAnalysis{}).
			Where("analyzer = 'batch-analyzer'").Count(&analysisCount).Error
		Expect(err).To(BeNil())
		Expect(analysisCount).To(Equal(int64(0)), "all batch config analysis records should be deleted")
	})

	ginkgo.AfterAll(func() {
		// Cleanup: Delete all test data

		// Delete component first (has foreign key to config_items)
		DefaultContext.DB().Where("name = ?", "protected-component").Delete(&models.Component{})

		// Delete related records for any remaining config items
		DefaultContext.DB().Where("analyzer IN (?)", []string{"test-analyzer", "batch-analyzer"}).Delete(&models.ConfigAnalysis{})
		DefaultContext.DB().Where("change_type = ?", "test-change").Delete(&models.ConfigChange{})
		DefaultContext.DB().Where("relation = ?", "test-relation").Delete(&models.ConfigRelationship{})

		// Delete config items
		DefaultContext.DB().Where("tags->>'test' = ?", "delete-old-config-items").Delete(&models.ConfigItem{})
		DefaultContext.DB().Where("tags->>'test' = ?", "delete-old-config-items-batch").Delete(&models.ConfigItem{})
	})
})
