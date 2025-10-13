package tests

import (
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
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

var _ = ginkgo.Describe("Permission Error Reset Trigger", ginkgo.Ordered, func() {
	var permission *models.Permission

	ginkgo.BeforeAll(func() {
		permission = &models.Permission{
			ID:             uuid.New(),
			Name:           "test-permission-trigger",
			Subject:        "test-group",
			SubjectType:    models.PermissionSubjectTypeGroup,
			Action:         "read",
			ObjectSelector: types.JSON(`{"playbooks": [{"name": "loki-logs"}]}`),
			Error:          lo.ToPtr("invalid selector error"),
			Source:         "UI",
		}
		err := DefaultContext.DB().Create(permission).Error
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		err := DefaultContext.DB().Delete(permission).Error
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.It("should reset error when object_selector changes", func() {
		// Verify initial error exists
		var fetched models.Permission
		err := DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).ToNot(BeNil())
		Expect(*fetched.Error).To(Equal("invalid selector error"))

		// Update object_selector
		newSelector := types.JSON(`{"playbooks": [{"name": "prometheus-metrics"}]}`)
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", newSelector).Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error was reset to NULL
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).To(BeNil())
		Expect(fetched.ObjectSelector).To(Equal(newSelector))
	})

	ginkgo.It("should NOT reset error when object_selector remains unchanged", func() {
		// Set an error
		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("error", "another error").Error
		Expect(err).ToNot(HaveOccurred())

		// Update a different field (action)
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("action", "write").Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error still exists
		var fetched models.Permission
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).ToNot(BeNil())
		Expect(*fetched.Error).To(Equal("another error"))
		Expect(fetched.Action).To(Equal("write"))
	})

	ginkgo.It("should reset error when object_selector changes from non-null to different value", func() {
		// First, set the object_selector
		selector1 := types.JSON(`{"config": [{"type": "Kubernetes::Pod"}]}`)
		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", selector1).Error
		Expect(err).ToNot(HaveOccurred())

		// Then set an error separately
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("error", "test error").Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error exists
		var fetched models.Permission
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).ToNot(BeNil())

		// Change object_selector to a different value
		selector2 := types.JSON(`{"config": [{"type": "Kubernetes::Node"}]}`)
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", selector2).Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error was reset
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).To(BeNil())
		Expect(fetched.ObjectSelector).To(Equal(selector2))
	})

	ginkgo.It("should treat semantically identical JSONB with different whitespace as equal", func() {
		// Set object_selector with compact format
		compactSelector := types.JSON(`{"playbooks": [{"name": "test-playbook"}]}`)
		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", compactSelector).Error
		Expect(err).ToNot(HaveOccurred())

		// Set an error
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("error", "whitespace test error").Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error exists
		var fetched models.Permission
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).ToNot(BeNil())

		// Update with same content but different whitespace
		// PostgreSQL normalizes JSONB, so this should be treated as the same value
		whitespaceSelector := types.JSON(`{"playbooks": [{"name":                "test-playbook"}]}`)
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", whitespaceSelector).Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error was NOT reset because JSONB values are semantically identical
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).ToNot(BeNil())
		Expect(*fetched.Error).To(Equal("whitespace test error"))
	})

	ginkgo.It("should reset error when JSONB content actually changes despite similar structure", func() {
		// Set initial selector
		selector1 := types.JSON(`{"playbooks": [{"namespace": "default"}]}`)
		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", selector1).Error
		Expect(err).ToNot(HaveOccurred())

		// Set an error
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("error", "content change test").Error
		Expect(err).ToNot(HaveOccurred())

		// Change to different namespace value
		selector2 := types.JSON(`{"playbooks": [{"namespace": "production"}]}`)
		err = DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", selector2).Error
		Expect(err).ToNot(HaveOccurred())

		// Verify error was reset because content changed
		var fetched models.Permission
		err = DefaultContext.DB().First(&fetched, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Error).To(BeNil())
	})
})
