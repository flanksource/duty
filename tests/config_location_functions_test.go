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
			Expect(results).To(ConsistOf([]uuid.UUID{dummy.KubernetesNodeA.ID}))
		})

		ginkgo.It("should find children based on external_id with prefix filter", func() {
			results, err := query.FindConfigChildrenIDsByLocation(DefaultContext, dummy.KubernetesNodeA.ID, "node://kubernetes")
			Expect(err).To(BeNil())
			Expect(results).To(ConsistOf([]uuid.UUID{dummy.LogisticsAPIPodConfig.ID, dummy.LogisticsUIPodConfig.ID}))
		})
	})

	ginkgo.Context("get_children_by_location function", func() {
		ginkgo.It("should find children based on external_id without prefix filter", func() {
			results, err := query.FindConfigChildrenByLocation(DefaultContext, dummy.EKSCluster.ID, "", false)
			Expect(err).To(BeNil())
			Expect(results).To(ConsistOf([]query.ConfigMinimal{
				{ID: dummy.KubernetesNodeA.ID, Name: *dummy.KubernetesNodeA.Name, Type: *dummy.KubernetesNodeA.Type},
			}))
		})

		ginkgo.It("should find children based on external_id with prefix filter", func() {
			results, err := query.FindConfigChildrenByLocation(DefaultContext, dummy.KubernetesNodeA.ID, "node://kubernetes", false)
			Expect(err).To(BeNil())
			Expect(results).To(ConsistOf([]query.ConfigMinimal{
				{ID: dummy.LogisticsAPIPodConfig.ID, Name: *dummy.LogisticsAPIPodConfig.Name, Type: *dummy.LogisticsAPIPodConfig.Type},
				{ID: dummy.LogisticsUIPodConfig.ID, Name: *dummy.LogisticsUIPodConfig.Name, Type: *dummy.LogisticsUIPodConfig.Type},
			}))
		})
	})

	ginkgo.Context("get_parent_ids_by_location function", func() {
		ginkgo.It("should find parent IDs based on external_id", func() {
			results, err := query.FindConfigParentIDsByLocation(DefaultContext, dummy.LogisticsAPIPodConfig.ID, "deployment://kubernetes")
			Expect(err).To(BeNil())
			Expect(results).To(ConsistOf([]uuid.UUID{dummy.LogisticsAPIDeployment.ID}))
		})

		ginkgo.It("should find parents based on external_id", func() {
			results, err := query.FindConfigParentsByLocation(DefaultContext, dummy.LogisticsAPIPodConfig.ID, "", false)
			Expect(err).To(BeNil())
			Expect(results).To(ConsistOf([]query.ConfigMinimal{
				{ID: dummy.KubernetesCluster.ID, Name: *dummy.KubernetesCluster.Name, Type: *dummy.KubernetesCluster.Type},
				{ID: dummy.KubernetesNodeA.ID, Name: *dummy.KubernetesNodeA.Name, Type: *dummy.KubernetesNodeA.Type},
				{ID: dummy.LogisticsAPIDeployment.ID, Name: *dummy.LogisticsAPIDeployment.Name, Type: *dummy.LogisticsAPIDeployment.Type},
				{ID: dummy.LogisticsAPIReplicaSet.ID, Name: *dummy.LogisticsAPIReplicaSet.Name, Type: *dummy.LogisticsAPIReplicaSet.Type},
				{ID: dummy.MissionControlNamespace.ID, Name: *dummy.MissionControlNamespace.Name, Type: *dummy.MissionControlNamespace.Type},
			}))
		})
	})
})
