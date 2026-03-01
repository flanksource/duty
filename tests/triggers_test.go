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
		DefaultContext.DB().Where("event_id = ?", ci.ID.String()).Delete(&models.Event{})
		DefaultContext.DB().Delete(ci)
	})

	ginkgo.BeforeEach(func() {
		DefaultContext.DB().Where("event_id = ?", ci.ID.String()).Delete(&models.Event{})
	})

	ginkgo.It("should NOT trigger events when inserting as healthy", func() {
		var events []models.Event
		err := DefaultContext.DB().Where("event_id = ?", ci.ID.String()).Find(&events).Error
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
		err = DefaultContext.DB().Where("event_id = ? AND name = ?", ci.ID.String(), "config.degraded").Find(&events).Error
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
		err = DefaultContext.DB().Where("event_id = ? AND name = ?", ci.ID.String(), "config.warning").Find(&events).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(events).To(HaveLen(1))
	})
})

var _ = ginkgo.Describe("Permission Triggers", ginkgo.Ordered, func() {
	var permission *models.Permission

	updateSelector := func(selector types.JSON) {
		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("object_selector", selector).Error
		Expect(err).ToNot(HaveOccurred())
	}

	setError := func(errMsg string) {
		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("error", errMsg).Error
		Expect(err).ToNot(HaveOccurred())
	}

	getPermission := func() models.Permission {
		var p models.Permission
		err := DefaultContext.DB().First(&p, "id = ?", permission.ID).Error
		Expect(err).ToNot(HaveOccurred())
		return p
	}

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
		p := getPermission()
		Expect(p.Error).ToNot(BeNil())
		Expect(*p.Error).To(Equal("invalid selector error"))

		updateSelector(types.JSON(`{"playbooks": [{"name": "prometheus-metrics"}]}`))

		p = getPermission()
		Expect(p.Error).To(BeNil())
	})

	ginkgo.It("should NOT reset error when object_selector remains unchanged", func() {
		setError("another error")

		err := DefaultContext.DB().Model(&models.Permission{}).
			Where("id = ?", permission.ID).
			Update("action", "write").Error
		Expect(err).ToNot(HaveOccurred())

		p := getPermission()
		Expect(p.Error).ToNot(BeNil())
		Expect(*p.Error).To(Equal("another error"))
	})

	ginkgo.It("should reset error when selector value changes", func() {
		updateSelector(types.JSON(`{"config": [{"type": "Kubernetes::Pod"}]}`))
		setError("test error")

		p := getPermission()
		Expect(p.Error).ToNot(BeNil())

		updateSelector(types.JSON(`{"config": [{"type": "Kubernetes::Node"}]}`))

		p = getPermission()
		Expect(p.Error).To(BeNil())
	})

	ginkgo.It("should treat JSONB with different whitespace as equal", func() {
		updateSelector(types.JSON(`{"playbooks": [{"name": "test-playbook"}]}`))
		setError("whitespace test")

		p := getPermission()
		Expect(p.Error).ToNot(BeNil())

		updateSelector(types.JSON(`{"playbooks": [{"name":                "test-playbook"}]}`))

		p = getPermission()
		Expect(p.Error).ToNot(BeNil())
		Expect(*p.Error).To(Equal("whitespace test"))
	})

	ginkgo.It("should reset error when JSONB content changes", func() {
		updateSelector(types.JSON(`{"playbooks": [{"namespace": "default"}]}`))
		setError("content test")

		updateSelector(types.JSON(`{"playbooks": [{"namespace": "production"}]}`))

		p := getPermission()
		Expect(p.Error).To(BeNil())
	})
})
