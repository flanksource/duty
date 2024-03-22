package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("SearchResourceSelectors", func() {
	ginkgo.It("should find all 3 resources", func() {
		items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
			Configs:    []types.ResourceSelector{{ID: dummy.EKSCluster.ID.String()}},
			Components: []types.ResourceSelector{{ID: dummy.Logistics.ID.String()}},
			Checks:     []types.ResourceSelector{{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}},
		})
		Expect(err).To(BeNil())

		expectation := []query.SelectedResource{
			{ID: dummy.EKSCluster.ID.String(), Name: lo.FromPtr(dummy.EKSCluster.Name), Type: query.SelectedResourceTypeConfig},
			{
				ID:   dummy.LogisticsAPIHealthHTTPCheck.ID.String(),
				Icon: dummy.LogisticsAPIHealthHTTPCheck.Icon,
				Name: dummy.LogisticsAPIHealthHTTPCheck.Name,
				Type: query.SelectedResourceTypeCheck,
			},
			{ID: dummy.Logistics.ID.String(), Icon: dummy.Logistics.Icon, Name: dummy.Logistics.Name, Type: query.SelectedResourceTypeComponent},
		}
		Expect(items).To(ConsistOf(expectation))
	})

	ginkgo.Context("field selector", ginkgo.Ordered, func() {
		ginkgo.It("Property lookup Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "region=us-west-2"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(items, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("Property lookup Not Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "region!=us-east-1"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(items, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("Property lookup Greater Than Query", func() {
			ginkgo.Skip("Implement for property lookup")
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "memory>5"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(items, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeA.ID.String(), dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("Property lookup Less Than Query", func() {
			ginkgo.Skip("Implement for property lookup")
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "memory<50"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(items, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "config_class in (Cluster)"}},
			})
			Expect(err).To(BeNil())

			Expect(len(items)).To(Equal(2), "should have returned 2 for EKS and Kubernetes Cluster")
		})

		ginkgo.It("NOT IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "config_class notin (Node,Deployment,Database,Pod,Cluster)"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2), "should have returned 2 for the Virtual Machine configs")
		})
	})

	ginkgo.Context("Label selector", func() {
		ginkgo.It("Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry=enabled,environment=production"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			Expect(items[0].ID).To(Equal(dummy.EKSCluster.ID.String()))
		})

		ginkgo.It("Not Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry=enabled,environment!=production"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			Expect(items[0].ID).To(Equal(dummy.KubernetesCluster.ID.String()))
		})

		ginkgo.It("IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "app in (frontend,backend)"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2))
			ids := lo.Map(items, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.EC2InstanceA.ID.String(), dummy.EC2InstanceB.ID.String()}))
		})

		ginkgo.It("NOT IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "app notin (frontend,logistics)"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ids := lo.Map(items, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.EC2InstanceA.ID.String()}))
		})
	})
})
