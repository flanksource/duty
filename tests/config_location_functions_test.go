package tests

import (
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = ginkgo.Describe("Config Location Functions", ginkgo.Ordered, func() {
	ginkgo.Context("get_children_id_by_location function", func() {
		ginkgo.It("should find children based on external_id without prefix filter", func() {
			results, err := query.FindConfigChildrenIDsByLocation(DefaultContext, dummy.EKSCluster.ID, "")
			Expect(err).To(BeNil())
			Expect(results).To(ContainElements([]uuid.UUID{dummy.EKSCluster.ID, dummy.KubernetesNodeA.ID}))
		})

		ginkgo.It("should find children based on external_id with prefix filter", func() {
			results, err := query.FindConfigChildrenIDsByLocation(DefaultContext, dummy.KubernetesNodeA.ID, "node://kubernetes")
			Expect(err).To(BeNil())
			Expect(results).To(ContainElements([]uuid.UUID{dummy.LogisticsAPIPodConfig.ID}))
		})
	})

	ginkgo.Context("get_children_by_location function", func() {
		ginkgo.It("should find children based on external_id without prefix filter", func() {
			results, err := query.FindConfigChildrenByLocation(DefaultContext, dummy.EKSCluster.ID, "", false)
			Expect(err).To(BeNil())
			Expect(results).To(ContainElements([]query.ChildByLocation{
				{ID: dummy.EKSCluster.ID, Name: *dummy.EKSCluster.Name, Type: *dummy.EKSCluster.Type},
				{ID: dummy.KubernetesNodeA.ID, Name: *dummy.KubernetesNodeA.Name, Type: *dummy.KubernetesNodeA.Type},
			}))
		})

		ginkgo.It("should find children based on external_id with prefix filter", func() {
			results, err := query.FindConfigChildrenByLocation(DefaultContext, dummy.KubernetesNodeA.ID, "node://kubernetes", false)
			Expect(err).To(BeNil())
			Expect(results).To(ContainElements([]query.ChildByLocation{
				{ID: dummy.LogisticsAPIPodConfig.ID, Name: *dummy.LogisticsAPIPodConfig.Name, Type: *dummy.LogisticsAPIPodConfig.Type},
			}))
		})
	})
})
