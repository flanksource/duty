package tests

import (
	"time"

	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Soft deleted components", ginkgo.Ordered, func() {
	var softDeletedComponents []models.Component

	ginkgo.It("should populated dummy deleted components", func() {
		data := dummy.GenerateDynamicDummyData(DefaultContext.DB())
		for i := range data.Components {
			data.Components[i].AgentID = uuid.Nil

			if i == 0 {
				data.Components[i].DeletedAt = lo.ToPtr(dummy.CurrentTime.Add(-10 * time.Minute))
				continue
			}

			data.Components[i].DeletedAt = lo.ToPtr(dummy.CurrentTime.Add(-time.Hour * 24 * 7))
		}

		softDeletedComponents = data.Components
		err := DefaultContext.DB().Create(&softDeletedComponents).Error
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.It("should delete soft deleted components", func() {
		count, err := job.CleanupSoftDeletedComponents(DefaultContext, time.Hour*24)
		Expect(err).ToNot(HaveOccurred())

		Expect(count).To(Equal(len(softDeletedComponents) - 1))
	})

	ginkgo.It("should cleanup the newly added soft deleted components", func() {
		err := DefaultContext.DB().Delete(&softDeletedComponents).Error
		Expect(err).ToNot(HaveOccurred())
	})
})
