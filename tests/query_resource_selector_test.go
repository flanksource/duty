package tests

import (
	"fmt"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
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
				Configs:    []types.ResourceSelector{{Health: types.MatchExpression(models.HealthHealthy)}},
				Components: []types.ResourceSelector{{Health: types.MatchExpression(models.HealthHealthy)}},
				Checks:     []types.ResourceSelector{{Health: types.MatchExpression(models.HealthHealthy)}},
			},
			Components: []models.Component{dummy.Logistics},
			Checks:     []models.Check{dummy.LogisticsAPIHealthHTTPCheck, dummy.LogisticsAPIHomeHTTPCheck},
			Configs:    []models.ConfigItem{dummy.KubernetesNodeAKSPool1, dummy.KubernetesNodeA, dummy.KubernetesNodeB},
		},
		{
			description: "name prefix | components",
			query: query.SearchResourcesRequest{
				Components: []types.ResourceSelector{{Search: "logistics-*", Types: []string{"Application"}}},
			},
			Components: []models.Component{dummy.LogisticsAPI, dummy.LogisticsUI, dummy.LogisticsWorker},
		},
		{
			description: "name prefix | checks",
			query: query.SearchResourcesRequest{
				Checks: []types.ResourceSelector{{Search: "logistics-*", Types: []string{"http"}}},
			},
			Checks: []models.Check{dummy.LogisticsAPIHomeHTTPCheck, dummy.LogisticsAPIHealthHTTPCheck},
		},
		{
			description: "name prefix | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Search: "node*"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeA, dummy.KubernetesNodeB},
		},
		{
			description: "name prefix with label selector",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Search: "node*)", LabelSelector: "region=us-west-2"}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeB},
		},
		// TODO: Currently search does not support labels/tags
		//{
		//description: "tag prefix - eg #1",
		//query: query.SearchResourcesRequest{
		//Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "aws*"}},
		//},
		//Configs: []models.ConfigItem{dummy.EKSCluster},
		//},
		//{
		//description: "tag prefix - eg #2",
		//query: query.SearchResourcesRequest{
		//Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "demo*"}},
		//},
		//Configs: []models.ConfigItem{dummy.KubernetesCluster},
		//},
		//{
		//description: "label prefix - eg #1",
		//query: query.SearchResourcesRequest{
		//Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "prod*"}},
		//},
		//Configs: []models.ConfigItem{dummy.EKSCluster},
		//},
		//{
		//description: "label prefix - eg #2",
		//query: query.SearchResourcesRequest{
		//Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "develop*"}},
		//},
		//Configs: []models.ConfigItem{dummy.KubernetesCluster},
		//},
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

var _ = ginkgo.Describe("Resoure Selector with PEG", ginkgo.Ordered, func() {
	ginkgo.BeforeAll(func() {
		_ = query.SyncConfigCache(DefaultContext)
	})

	// = , != , item in list, item not in list, prefix, suffix, date operations (created_at, updated_at), agent query
	testData := []struct {
		description string
		query       string
		expectedIDs []uuid.UUID
		resource    string
	}{
		{
			description: "config item direct query without quotes",
			query:       `node-b`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeB.ID},
			resource:    "config",
		},
		{
			description: "config item direct query with quotes",
			query:       `"node-b"`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeB.ID},
			resource:    "config",
		},
		{
			description: "config item direct query no match",
			query:       `unknown-name-config`,
			expectedIDs: []uuid.UUID{},
			resource:    "config",
		},
		{
			description: "config item name query no match",
			query:       `name=unknown-name-config`,
			expectedIDs: []uuid.UUID{},
			resource:    "config",
		},
		{
			description: "config item query with :: in string",
			query:       `name=node-b type=Kubernetes::Node`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeB.ID},
			resource:    "config",
		},
		{
			description: "config item query with quotes",
			query:       `name="node-b" type="Kubernetes::Node"`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeB.ID},
			resource:    "config",
		},
		{
			description: "config item not equal to query",
			query:       `name!="node-b" type="Kubernetes::Node"`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeA.ID, dummy.KubernetesNodeAKSPool1.ID},
			resource:    "config",
		},
		{
			description: "component query",
			query:       `type=Application`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPI.ID, dummy.LogisticsUI.ID, dummy.LogisticsWorker.ID, dummy.KustomizeFluxComponent.ID},
			resource:    "component",
		},
		{
			description: "component in query",
			query:       `type=Application,Gap`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPI.ID, dummy.LogisticsUI.ID, dummy.LogisticsWorker.ID, dummy.KustomizeFluxComponent.ID},
			resource:    "component",
		},
		{
			description: "component agent query",
			query:       `agent="GCP"`,
			expectedIDs: []uuid.UUID{dummy.PaymentsAPI.ID},
			resource:    "component",
		},
		{
			description: "component agent_id query",
			query:       fmt.Sprintf(`agent_id="%s"`, dummy.GCPAgent.ID.String()),
			expectedIDs: []uuid.UUID{dummy.PaymentsAPI.ID},
			resource:    "component",
		},
		{
			description: "component created_at query",
			query:       `created_at>2023-01-01`,
			expectedIDs: []uuid.UUID{dummy.FluxComponent.ID},
			resource:    "component",
		},
		{
			description: "component created_at query with quotes",
			query:       `created_at>"2023-01-01"`,
			expectedIDs: []uuid.UUID{dummy.FluxComponent.ID},
			resource:    "component",
		},
		{
			// This tests now-t feature of date time
			// If this test fails, adjust relative time in query
			// for the expected result
			description: "component created_at now query",
			query:       `created_at>now-2y`,
			expectedIDs: []uuid.UUID{dummy.FluxComponent.ID},
			resource:    "component",
		},
		{
			// This tests now-t feature of date time
			// If this test fails, adjust relative time in query
			// for the expected result
			description: "component created_at now query with quotes",
			query:       `created_at>"now-2y"`,
			expectedIDs: []uuid.UUID{dummy.FluxComponent.ID},
			resource:    "component",
		},
		{
			description: "component prefix and suffix query",
			query:       `type=Kubernetes* type="*Pod"`,
			expectedIDs: []uuid.UUID{dummy.LogisticsUIPod.ID, dummy.LogisticsAPIPod.ID, dummy.LogisticsWorkerPod.ID},
			resource:    "component",
		},
		{
			description: "component complex not in query",
			query:       `type!="Application,Entity,Database,Kubernetes*,Flux*"`, // This covers all types in dummy components
			expectedIDs: []uuid.UUID{},
			resource:    "component",
		},
		{
			description: "config soft and limit query",
			query:       `name=node-* type=Kubernetes::Node limit=1 sort=name`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeA.ID},
			resource:    "config",
		},
		{
			description: "config json query",
			query:       `config.metadata.name=node-a`,
			expectedIDs: []uuid.UUID{dummy.KubernetesNodeA.ID},
			resource:    "config",
		},
		{
			description: "config json integer query",
			query:       `config.spec.replicas=3`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPIDeployment.ID},
			resource:    "config",
		},
		{
			description: "config labels query",
			query:       `labels.account=flanksource labels.environment=production`,
			expectedIDs: []uuid.UUID{dummy.EKSCluster.ID, dummy.EC2InstanceB.ID},
			resource:    "config",
		},
		{
			description: "config array query",
			query:       `config.spec.template.spec.containers[0].name=logistics-api`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPIDeployment.ID},
			resource:    "config",
		},
		{
			description: "config array query with integer matching",
			query:       `config.spec.template.spec.containers[0].ports[0].containerPort=80`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPIDeployment.ID},
			resource:    "config",
		},
	}

	fmap := map[string]func(context.Context, int, ...types.ResourceSelector) ([]uuid.UUID, error){
		"config":    query.FindConfigIDsByResourceSelector,
		"component": query.FindComponentIDs,
		"checks":    query.FindCheckIDs,
	}

	uuidSliceToString := func(uuids []uuid.UUID) []string {
		return lo.Map(uuids, func(item uuid.UUID, _ int) string { return item.String() })
	}

	for _, tt := range testData {
		ginkgo.It(tt.description, func() {
			f, ok := fmap[tt.resource]
			Expect(ok).To(BeTrue())
			ids, err := f(DefaultContext, -1, types.ResourceSelector{Search: tt.query})
			Expect(err).To(BeNil())
			// We convert to strings slice for readable output
			Expect(uuidSliceToString(ids)).To(ConsistOf(uuidSliceToString(tt.expectedIDs)))
		})
	}
})
