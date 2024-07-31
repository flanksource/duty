package tests

import (
	"fmt"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("SearchResourceSelectors", func() {
	ginkgo.It("should find all 3 resources", func() {
		response, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
			Configs:    []types.ResourceSelector{{ID: dummy.EKSCluster.ID.String()}},
			Components: []types.ResourceSelector{{ID: dummy.Logistics.ID.String()}},
			Checks:     []types.ResourceSelector{{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}},
		})
		Expect(err).To(BeNil())

		expectation := &query.SearchResourcesResponse{
			Configs: []query.SelectedResource{{
				ID:        dummy.EKSCluster.ID.String(),
				Agent:     dummy.EKSCluster.AgentID.String(),
				Tags:      dummy.EKSCluster.Tags,
				Name:      lo.FromPtr(dummy.EKSCluster.Name),
				Namespace: dummy.EKSCluster.GetNamespace(),
				Type:      lo.FromPtr(dummy.EKSCluster.Type),
			}},
			Checks: []query.SelectedResource{{
				ID:        dummy.LogisticsAPIHealthHTTPCheck.ID.String(),
				Agent:     dummy.LogisticsAPIHealthHTTPCheck.AgentID.String(),
				Icon:      dummy.LogisticsAPIHealthHTTPCheck.Icon,
				Tags:      dummy.LogisticsAPIHealthHTTPCheck.Labels,
				Name:      dummy.LogisticsAPIHealthHTTPCheck.Name,
				Namespace: dummy.LogisticsAPIHealthHTTPCheck.Namespace,
				Type:      dummy.LogisticsAPIHealthHTTPCheck.Type,
			}},
			Components: []query.SelectedResource{{
				ID:        dummy.Logistics.ID.String(),
				Agent:     dummy.Logistics.AgentID.String(),
				Icon:      dummy.Logistics.Icon,
				Tags:      dummy.Logistics.Labels,
				Name:      dummy.Logistics.Name,
				Namespace: dummy.Logistics.Namespace,
				Type:      dummy.Logistics.Type,
			}},
		}
		Expect(response).To(Equal(expectation))
	})

	ginkgo.Context("field selector", ginkgo.Ordered, func() {
		ginkgo.It("Property lookup Equals Query", func() {
			response, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "region=us-west-2"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(response.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("Property lookup Not Equals Query", func() {
			response, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "region!=us-east-1", TagSelector: "cluster=aws"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(response.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("Property lookup Greater Than Query", func() {
			ginkgo.Skip("Implement for property lookup")
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "memory>5"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(items.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeA.ID.String(), dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("Property lookup Less Than Query", func() {
			ginkgo.Skip("Implement for property lookup")
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "memory<50"}},
			})
			Expect(err).To(BeNil())
			ids := lo.Map(items.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.KubernetesNodeB.ID.String()}))
		})

		ginkgo.It("IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "config_class in (Cluster)"}},
			})
			Expect(err).To(BeNil())

			Expect(len(items.Configs)).To(Equal(2), "should have returned 2 for EKS and Kubernetes Cluster")
		})

		ginkgo.It("NOT IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "config_class notin (Node,Deployment,Database,Pod,Cluster)"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(3), "should have returned 3 for the Virtual Machine configs")
		})
	})

	ginkgo.Context("Tag selector", func() {
		ginkgo.It("Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Types: []string{"EKS::Cluster"}, TagSelector: "cluster=aws,account=flanksource"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(1))
			Expect(items.Configs[0].ID).To(Equal(dummy.EKSCluster.ID.String()))
		})

		ginkgo.It("Not Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{TagSelector: "cluster!=aws", Types: []string{"Kubernetes::Cluster"}}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(1))
			Expect(items.Configs[0].ID).To(Equal(dummy.KubernetesCluster.ID.String()))
		})
	})

	ginkgo.Context("Label selector", func() {
		ginkgo.It("Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry=enabled,environment=production"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(1))
			Expect(items.Configs[0].ID).To(Equal(dummy.EKSCluster.ID.String()))
		})

		ginkgo.It("Not Equals Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry=enabled,environment!=production"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(1))
			Expect(items.Configs[0].ID).To(Equal(dummy.KubernetesCluster.ID.String()))
		})

		ginkgo.It("IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "app in (frontend,backend)"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(2))
			ids := lo.Map(items.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.EC2InstanceA.ID.String(), dummy.EC2InstanceB.ID.String()}))
		})

		ginkgo.It("NOT IN Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "app notin (frontend,logistics)"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(1))
			ids := lo.Map(items.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.EC2InstanceA.ID.String()}))
		})

		ginkgo.It("Exists Query", func() {
			items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry,environment"}},
			})
			Expect(err).To(BeNil())
			Expect(len(items.Configs)).To(Equal(2))
			ids := lo.Map(items.Configs, func(item query.SelectedResource, _ int) string { return item.ID })
			Expect(ids).To(ConsistOf([]string{dummy.EKSCluster.ID.String(), dummy.KubernetesCluster.ID.String()}))
		})
	})

	ginkgo.Context("search query", ginkgo.Ordered, func() {
		testData := []struct {
			description string
			query       query.SearchResourcesRequest
			Configs     []uuid.UUID
			Components  []uuid.UUID
			Checks      []uuid.UUID
		}{
			{
				description: "name prefix | components",
				query: query.SearchResourcesRequest{
					Components: []types.ResourceSelector{{Search: "logistics-", Types: []string{"Application"}}},
				},
				Components: []uuid.UUID{dummy.LogisticsAPI.ID, dummy.LogisticsUI.ID, dummy.LogisticsWorker.ID},
			},
			{
				description: "name prefix | checks",
				query: query.SearchResourcesRequest{
					Checks: []types.ResourceSelector{{Search: "logistics-", Types: []string{"http"}}},
				},
				Checks: []uuid.UUID{dummy.LogisticsAPIHomeHTTPCheck.ID, dummy.LogisticsAPIHealthHTTPCheck.ID},
			},
			{
				description: "name prefix | configs",
				query: query.SearchResourcesRequest{
					Configs: []types.ResourceSelector{{Search: "node"}},
				},
				Configs: []uuid.UUID{dummy.KubernetesNodeA.ID, dummy.KubernetesNodeB.ID},
			},
			{
				description: "name prefix with label selector",
				query: query.SearchResourcesRequest{
					Configs: []types.ResourceSelector{{Search: "node", LabelSelector: "region=us-west-2"}},
				},
				Configs: []uuid.UUID{dummy.KubernetesNodeB.ID},
			},
			{
				description: "tag prefix - eg #1",
				query: query.SearchResourcesRequest{
					Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "aws"}},
				},
				Configs: []uuid.UUID{dummy.EKSCluster.ID},
			},
			{
				description: "tag prefix - eg #2",
				query: query.SearchResourcesRequest{
					Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "demo"}},
				},
				Configs: []uuid.UUID{dummy.KubernetesCluster.ID},
			},
			{
				description: "label prefix - eg #1",
				query: query.SearchResourcesRequest{
					Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "prod"}},
				},
				Configs: []uuid.UUID{dummy.EKSCluster.ID},
			},
			{
				description: "label prefix - eg #2",
				query: query.SearchResourcesRequest{
					Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "develop"}},
				},
				Configs: []uuid.UUID{dummy.KubernetesCluster.ID},
			},
		}

		for _, test := range testData {
			ginkgo.It(test.description, func() {
				items, err := query.SearchResources(DefaultContext, test.query)
				Expect(err).To(BeNil())

				{
					// configs
					Expect(len(items.Configs)).To(Equal(len(test.Configs)))
					ids := lo.Map(items.Configs, func(item query.SelectedResource, _ int) uuid.UUID { return uuid.MustParse(item.ID) })
					Expect(ids).To(ConsistOf(test.Configs))
				}

				{
					// components
					Expect(len(items.Components)).To(Equal(len(test.Components)))
					ids := lo.Map(items.Components, func(item query.SelectedResource, _ int) uuid.UUID { return uuid.MustParse(item.ID) })
					Expect(ids).To(ConsistOf(test.Components))
				}

				{
					// checks
					Expect(len(items.Checks)).To(Equal(len(test.Checks)))
					ids := lo.Map(items.Checks, func(item query.SelectedResource, _ int) uuid.UUID { return uuid.MustParse(item.ID) })
					Expect(ids).To(ConsistOf(test.Checks))
				}
			})
		}
	})
})
