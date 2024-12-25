package tests

import (
	"fmt"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

func ExpectSearch(q query.SearchResourcesRequest) *query.SearchResourcesResponse {
	response, err := query.SearchResources(DefaultContext, q)
	Expect(err).To(BeNil())
	Expect(response).ToNot(BeNil())
	return response
}

var _ = ginkgo.Describe("SearchResourceSelectors", func() {
	testData := []struct {
		description string
		query       query.SearchResourcesRequest
		Configs     []models.ConfigItem
		Components  []models.Component
		Checks      []models.Check
	}{
		{
			description: "id",
			query: query.SearchResourcesRequest{
				Configs:    []types.ResourceSelector{{ID: dummy.EKSCluster.ID.String()}},
				Components: []types.ResourceSelector{{ID: dummy.Logistics.ID.String()}},
				Checks:     []types.ResourceSelector{{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}},
			},
			Components: []models.Component{dummy.Logistics},
			Checks:     []models.Check{dummy.LogisticsAPIHealthHTTPCheck},
			Configs:    []models.ConfigItem{dummy.EKSCluster},
		},
		{
			description: "health",
			query: query.SearchResourcesRequest{
				Configs:    []types.ResourceSelector{{Healths: []string{string(models.HealthHealthy)}}},
				Components: []types.ResourceSelector{{Healths: []string{string(models.HealthHealthy)}}},
				Checks:     []types.ResourceSelector{{Healths: []string{string(models.HealthHealthy)}}},
			},
			Components: []models.Component{dummy.Logistics},
			Checks:     []models.Check{dummy.LogisticsAPIHealthHTTPCheck, dummy.LogisticsAPIHomeHTTPCheck},
			Configs:    []models.ConfigItem{dummy.KubernetesNodeAKSPool1, dummy.KubernetesNodeA, dummy.KubernetesNodeB},
		},
		{
			description: "name prefix | components",
			query: query.SearchResourcesRequest{
				Components: []types.ResourceSelector{{Search: "logistics-", Types: []string{"Application"}}},
			},
			Components: []models.Component{dummy.LogisticsAPI, dummy.LogisticsUI, dummy.LogisticsWorker},
		},
		{
			description: "name prefix | checks",
			query: query.SearchResourcesRequest{
				Checks: []types.ResourceSelector{{Search: "logistics-", Types: []string{"http"}}},
			},
			Checks: []models.Check{dummy.LogisticsAPIHomeHTTPCheck, dummy.LogisticsAPIHealthHTTPCheck},
		},
		{
			description: "name prefix | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Search: "node"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeA, dummy.KubernetesNodeB},
		},
		{
			description: "name prefix with label selector",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Search: "node", LabelSelector: "region=us-west-2"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeB},
		},
		{
			description: "tag prefix - eg #1",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "aws"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster},
		},
		{
			description: "tag prefix - eg #2",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "demo"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesCluster},
		},
		{
			description: "label prefix - eg #1",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "prod"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster},
		},
		{
			description: "label prefix - eg #2",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "develop"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesCluster},
		},
		{
			description: "labels | Equals Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry=enabled,environment=production"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster},
		},
		{
			description: "labels | Not Equals Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry=enabled,environment!=production"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesCluster},
		},
		{
			description: "labels | IN Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "app in (frontend,backend)"}},
			},
			Configs: []models.ConfigItem{dummy.EC2InstanceA, dummy.EC2InstanceB},
		},
		{
			description: "labels | NOT IN Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "app notin (frontend,logistics)"}},
			},
			Configs: []models.ConfigItem{dummy.EC2InstanceA},
		},
		{
			description: "labels | Exists Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "telemetry,environment"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster, dummy.KubernetesCluster},
		},
		{
			description: "field selector | Property lookup Equals Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "region=us-west-2"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeB},
		},
		{
			description: "field selector | Property lookup Not Equals Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "region!=us-east-1", TagSelector: "cluster=aws"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeB},
		},
		{
			description: "field selector | Property lookup Greater Than Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "memory>5"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeA, dummy.KubernetesNodeB},
		},
		{
			description: "field selector | Property lookup Less Than Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "memory<50"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeB},
		},
		{
			description: "field selector | IN Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "config_class in (Cluster)"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster, dummy.KubernetesCluster},
		},
		{
			description: "field selector | NOT IN Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{FieldSelector: "config_class notin (Node,Deployment,Database,Pod,Cluster)"}},
			},
			Configs: []models.ConfigItem{dummy.EC2InstanceA, dummy.EC2InstanceB},
		},
		{
			description: "field selector | Tag selector Equals Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Types: []string{"EKS::Cluster"}, TagSelector: "cluster=aws,account=flanksource"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster},
		},
		{
			description: "field selector | Tag selector Not Equals Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{TagSelector: "cluster!=aws", Types: []string{"Kubernetes::Cluster"}}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesCluster},
		},
	}

	ginkgo.Describe("search", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			_ = query.SyncConfigCache(DefaultContext)
		})

		ginkgo.Context("query", func() {
			for _, test := range testData {
				ginkgo.It(test.description, func() {
					items, err := query.SearchResources(DefaultContext, test.query)
					Expect(err).To(BeNil())
					Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Configs...)), "should contain configs")
					Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Components...)), "should contain components")
					Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Checks...)), "should contain checks")
				})
			}
		})
	})
})

var _ = ginkgo.Describe("Resoure Selector limits", ginkgo.Ordered, func() {
	ginkgo.BeforeAll(func() {
		_ = query.SyncConfigCache(DefaultContext)
	})

	ginkgo.Context("It should return the fixed page size for configs", func() {
		for limit := 1; limit < 3; limit++ {
			ginkgo.It(fmt.Sprintf("should work with %d page size", limit), func() {
				items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
					Limit:   limit,
					Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassNode)}},
				})

				Expect(err).To(BeNil())
				Expect(limit).To(Equal(len(items.Configs)))
			})
		}
	})

	ginkgo.Context("It should return the fixed page size for components", func() {
		for limit := 1; limit < 5; limit++ {
			ginkgo.It(fmt.Sprintf("should work with %d page size", limit), func() {
				items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
					Limit:      limit,
					Components: []types.ResourceSelector{{Types: []string{"Application"}}},
				})

				Expect(err).To(BeNil())
				Expect(limit).To(Equal(len(items.Components)))
			})
		}
	})

	ginkgo.Context("It should return the fixed page size for checks", func() {
		for limit := 1; limit < 3; limit++ {
			ginkgo.It(fmt.Sprintf("should work with %d page size", limit), func() {
				items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
					Limit:  limit,
					Checks: []types.ResourceSelector{{Types: []string{"http"}, Agent: "all"}},
				})

				Expect(err).To(BeNil())
				Expect(limit).To(Equal(len(items.Checks)))
			})
		}
	})

	ginkgo.Context("It should return the fixed page size for all types", func() {
		for pageSize := 1; pageSize < 3; pageSize++ {
			ginkgo.It(fmt.Sprintf("should work with %d page size", pageSize), func() {
				items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
					Limit:      pageSize,
					Configs:    []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassNode)}},
					Components: []types.ResourceSelector{{Types: []string{"Application"}}},
					Checks:     []types.ResourceSelector{{Types: []string{"http"}, Agent: "all"}},
				})

				Expect(err).To(BeNil())
				Expect(pageSize).To(Equal(len(items.Configs)))
				Expect(pageSize).To(Equal(len(items.Components)))
				Expect(pageSize).To(Equal(len(items.Checks)))
			})
		}
	})
})
