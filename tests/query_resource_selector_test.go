package tests

import (
	"fmt"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

var (
	eksClusterCatalogChange = models.CatalogChange{
		ID:            uuid.MustParse(dummy.EKSClusterCreateChange.ID),
		ConfigID:      uuid.MustParse(dummy.EKSClusterCreateChange.ConfigID),
		Name:          dummy.EKSCluster.Name,
		Type:          dummy.EKSCluster.Type,
		Tags:          dummy.EKSCluster.Tags,
		CreatedAt:     dummy.EKSClusterCreateChange.CreatedAt,
		Severity:      lo.ToPtr(string(dummy.EKSClusterCreateChange.Severity)),
		ChangeType:    dummy.EKSClusterCreateChange.ChangeType,
		Source:        &dummy.EKSClusterCreateChange.Source,
		Summary:       &dummy.EKSClusterCreateChange.Summary,
		Count:         dummy.EKSClusterCreateChange.Count,
		FirstObserved: dummy.EKSClusterCreateChange.FirstObserved,
		AgentID:       &dummy.EKSCluster.AgentID,
	}

	kubernetesNodeACatalogChange = models.CatalogChange{
		ID:            uuid.MustParse(dummy.KubernetesNodeAChange.ID),
		ConfigID:      uuid.MustParse(dummy.KubernetesNodeAChange.ConfigID),
		Name:          dummy.KubernetesNodeA.Name,
		Type:          dummy.KubernetesNodeA.Type,
		Tags:          dummy.KubernetesNodeA.Tags,
		CreatedAt:     dummy.KubernetesNodeAChange.CreatedAt,
		Severity:      lo.ToPtr(string(dummy.KubernetesNodeAChange.Severity)),
		ChangeType:    dummy.KubernetesNodeAChange.ChangeType,
		Source:        &dummy.KubernetesNodeAChange.Source,
		Summary:       &dummy.KubernetesNodeAChange.Summary,
		Count:         dummy.KubernetesNodeAChange.Count,
		FirstObserved: dummy.KubernetesNodeAChange.FirstObserved,
		AgentID:       &dummy.KubernetesNodeA.AgentID,
	}
)

var _ = ginkgo.Describe("SearchResourceSelectors", func() {
	testData := []struct {
		description   string
		query         query.SearchResourcesRequest
		Canaries      []models.Canary
		Configs       []models.ConfigItem
		Components    []models.Component
		Checks        []models.Check
		ConfigChanges []models.CatalogChange
		Playbooks     []models.Playbook
		Connections   []models.Connection
	}{
		{
			description: "id",
			query: query.SearchResourcesRequest{
				Configs:       []types.ResourceSelector{{ID: dummy.EKSCluster.ID.String()}},
				Components:    []types.ResourceSelector{{ID: dummy.Logistics.ID.String()}},
				Checks:        []types.ResourceSelector{{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}},
				ConfigChanges: []types.ResourceSelector{{ID: dummy.EKSClusterCreateChange.ID}},
			},
			Components:    []models.Component{dummy.Logistics},
			Checks:        []models.Check{dummy.LogisticsAPIHealthHTTPCheck},
			Configs:       []models.ConfigItem{dummy.EKSCluster},
			ConfigChanges: []models.CatalogChange{eksClusterCatalogChange},
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
			description: "namespace | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Namespace: "missioncontrol", Types: []string{*dummy.LogisticsDBRDS.Type}}},
			},
			Configs: []models.ConfigItem{dummy.LogisticsDBRDS},
		},
		{
			description: "name prefix | configs | By Field",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Name: "node*"}},
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
		{
			description: "type exact match",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Types: []string{"Kubernetes::Cluster"}}},
			},
			Configs: []models.ConfigItem{
				dummy.KubernetesCluster,
			},
		},
		{
			description: "type suffix (implicit)", // if the user searches for type=POD, we must match Kubernetes::Pod
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Types: []string{"cluster"}}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesCluster},
		},
		{
			description: "type prefix | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Types: []string{"Logistics::DB*"}}},
			},
			Configs: []models.ConfigItem{dummy.LogisticsDBRDS},
		},
		{
			description: "type wildcard matching multiple types | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Types: []string{"Kubernetes::*"}}},
			},
			Configs: []models.ConfigItem{
				dummy.KubernetesCluster,
				dummy.KubernetesNodeA,
				dummy.KubernetesNodeB,
				dummy.KubernetesNodeAKSPool1,
				dummy.LogisticsAPIDeployment,
				dummy.LogisticsAPIReplicaSet,
				dummy.LogisticsAPIPodConfig,
				dummy.LogisticsUIDeployment,
				dummy.LogisticsUIReplicaSet,
				dummy.LogisticsUIPodConfig,
				dummy.LogisticsWorkerDeployment,
				dummy.MissionControlNamespace,
			},
		},
		{
			description: "status prefix | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Statuses: []string{"Run*"}}},
			},
			Configs: []models.ConfigItem{dummy.LogisticsAPIPodConfig},
		},
		{
			description: "health exclusion | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Health: "!healthy"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster, dummy.KubernetesCluster},
		},
		{
			description: "health inclusion | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Health: "unknown"}},
			},
			Configs: []models.ConfigItem{dummy.EKSCluster, dummy.KubernetesCluster},
		},
		{
			description: "health exclusion | checks",
			query: query.SearchResourcesRequest{
				Checks: []types.ResourceSelector{{Health: "!healthy"}},
			},
			Checks: []models.Check{dummy.LogisticsDBCheck},
		},
		{
			description: "namespace | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Namespace: "missioncontrol", Types: []string{*dummy.LogisticsDBRDS.Type}}},
			},
			Configs: []models.ConfigItem{dummy.LogisticsDBRDS},
		},
		{
			description: "config changes by multiple config types",
			query: query.SearchResourcesRequest{
				ConfigChanges: []types.ResourceSelector{{Types: []string{*dummy.EKSCluster.Type, *dummy.KubernetesNodeA.Type}}},
			},
			ConfigChanges: []models.CatalogChange{eksClusterCatalogChange, kubernetesNodeACatalogChange},
		},
		// TODO: Currently search does not support labels/tags
		// {
		// 	description: "tag prefix - eg #1",
		// 	Configs:     []models.ConfigItem{dummy.EKSCluster},
		// 	query: query.SearchResourcesRequest{
		// 		Configs: []types.ResourceSelector{
		// 			{
		// 				// FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster),
		// 				Search: "aws*",
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	description: "tag prefix - eg #2",
		// 	query: query.SearchResourcesRequest{
		// 		Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "demo*"}},
		// 	},
		// 	Configs: []models.ConfigItem{dummy.KubernetesCluster},
		// },
		// {
		// 	description: "label prefix - eg #1",
		// 	query: query.SearchResourcesRequest{
		// 		Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "prod*"}},
		// 	},
		// 	Configs: []models.ConfigItem{dummy.EKSCluster},
		// },
		// {
		// 	description: "label prefix - eg #2",
		// 	query: query.SearchResourcesRequest{
		// 		Configs: []types.ResourceSelector{{FieldSelector: fmt.Sprintf("config_class=%s", models.ConfigClassCluster), Search: "develop*"}},
		// 	},
		// 	Configs: []models.ConfigItem{dummy.KubernetesCluster},
		// },
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
			description: "labels | DoesNotExist Query",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{LabelSelector: "!storageprofile", Types: []string{"Kubernetes::Node"}}},
			},
			Configs: []models.ConfigItem{dummy.KubernetesNodeA},
		},
		{
			description: "search | field selector | prefix | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Search: "config_class=Virtual*"}},
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
		{
			description: "config changes by config type",
			query: query.SearchResourcesRequest{
				ConfigChanges: []types.ResourceSelector{{Types: []string{*dummy.EKSCluster.Type}}},
			},
			ConfigChanges: []models.CatalogChange{eksClusterCatalogChange},
		},
		{
			description: "config changes with limit",
			query: query.SearchResourcesRequest{
				Limit:         1,
				ConfigChanges: []types.ResourceSelector{{Types: []string{*dummy.EKSCluster.Type}}},
			},
			ConfigChanges: []models.CatalogChange{eksClusterCatalogChange},
		},
		{
			description: "playbook by id",
			query: query.SearchResourcesRequest{
				Playbooks: []types.ResourceSelector{{ID: dummy.EchoConfig.ID.String()}},
			},
			Playbooks: []models.Playbook{dummy.EchoConfig},
		},
		{
			description: "playbook by name and namespace",
			query: query.SearchResourcesRequest{
				Playbooks: []types.ResourceSelector{{Name: dummy.EchoConfig.Name, Namespace: dummy.EchoConfig.Namespace}},
			},
			Playbooks: []models.Playbook{dummy.EchoConfig},
		},
		{
			description: "connection by id",
			query: query.SearchResourcesRequest{
				Connections: []types.ResourceSelector{{ID: dummy.AWSConnection.ID.String()}},
			},
			Connections: []models.Connection{dummy.AWSConnection},
		},
		{
			description: "connection by type",
			query: query.SearchResourcesRequest{
				Connections: []types.ResourceSelector{{Types: []string{dummy.AWSConnection.Type}}},
			},
			Connections: []models.Connection{dummy.AWSConnection},
		},
		{
			description: "connection by namespace",
			query: query.SearchResourcesRequest{
				Connections: []types.ResourceSelector{{Namespace: dummy.PostgresConnection.Namespace}},
			},
			Connections: []models.Connection{dummy.PostgresConnection},
		},
		{
			description: "canary by id",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{ID: dummy.LogisticsAPICanary.ID.String()}},
			},
			Canaries: []models.Canary{dummy.LogisticsAPICanary},
		},
		{
			description: "canary by name",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{Name: dummy.LogisticsDBCanary.Name}},
			},
			Canaries: []models.Canary{dummy.LogisticsDBCanary},
		},
		{
			description: "canary by namespace",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{Namespace: "logistics"}},
			},
			Canaries: []models.Canary{dummy.LogisticsAPICanary, dummy.LogisticsDBCanary},
		},
		{
			description: "canary by name and namespace",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{Name: dummy.LogisticsAPICanary.Name, Namespace: dummy.LogisticsAPICanary.Namespace}},
			},
			Canaries: []models.Canary{dummy.LogisticsAPICanary},
		},
		{
			description: "canary by agent",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{Agent: dummy.GCPAgent.ID.String()}},
			},
			Canaries: []models.Canary{dummy.CartAPICanaryAgent},
		},
		{
			description: "canary with name prefix",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{Name: "dummy-logistics-*"}},
			},
			Canaries: []models.Canary{dummy.LogisticsAPICanary, dummy.LogisticsDBCanary},
		},
		{
			description: "multiple resource types including canaries",
			query: query.SearchResourcesRequest{
				Canaries: []types.ResourceSelector{{Namespace: "logistics"}},
				Checks:   []types.ResourceSelector{{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}},
				Configs:  []types.ResourceSelector{{ID: dummy.EKSCluster.ID.String()}},
			},
			Canaries: []models.Canary{dummy.LogisticsAPICanary, dummy.LogisticsDBCanary},
			Checks:   []models.Check{dummy.LogisticsAPIHealthHTTPCheck},
			Configs:  []models.ConfigItem{dummy.EKSCluster},
		},
	}

	ginkgo.Describe("search", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			Expect(query.SyncConfigCache(DefaultContext)).To(Succeed())
			Expect(query.PopulateAllTypesCache(DefaultContext)).To(Succeed())
		})

		for _, test := range testData {
			// if test.description != "type suffix (implicit)" {
			// 	continue
			// }

			ginkgo.It(test.description, func() {
				items, err := query.SearchResources(DefaultContext, test.query)
				Expect(err).To(BeNil())
				Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Canaries...)), "should contain canaries")
				Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Configs...)), "should contain configs")
				Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Components...)), "should contain components")
				Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Checks...)), "should contain checks")
				Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.ConfigChanges...)), "should contain config changes")
			})
		}
	})
})

var _ = ginkgo.Describe("Search Properties", ginkgo.Ordered, ginkgo.Pending, func() {
	ginkgo.BeforeAll(func() {
		Expect(query.SyncConfigCache(DefaultContext)).To(Succeed())
		Expect(query.PopulateAllTypesCache(DefaultContext)).To(Succeed())
	})

	testData := []struct {
		description   string
		query         query.SearchResourcesRequest
		Canaries      []models.Canary
		Configs       []models.ConfigItem
		Components    []models.Component
		Checks        []models.Check
		ConfigChanges []models.CatalogChange
		Playbooks     []models.Playbook
		Connections   []models.Connection
	}{
		{
			description: "field selector | Property lookup | configs",
			query: query.SearchResourcesRequest{
				Configs: []types.ResourceSelector{{Search: "properties.os=linux"}},
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
	}

	for _, test := range testData {
		if test.description != "field selector | Property lookup | configs" {
			continue
		}

		ginkgo.It(test.description, func() {
			items, err := query.SearchResources(DefaultContext, test.query)
			Expect(err).To(BeNil())
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Canaries...)), "should contain canaries")
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Configs...)), "should contain configs")
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Components...)), "should contain components")
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Checks...)), "should contain checks")
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.ConfigChanges...)), "should contain config changes")
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Playbooks...)), "should contain playbooks")
			Expect(items.GetIDs()).To(ContainElements(models.GetIDs(test.Connections...)), "should contain connections")
		})
	}
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
					Configs: []types.ResourceSelector{{Search: fmt.Sprintf("config_class=%s", models.ConfigClassNode)}},
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

	ginkgo.Context("It should return the fixed page size for canaries", func() {
		for limit := 1; limit < 3; limit++ {
			ginkgo.It(fmt.Sprintf("should work with %d page size", limit), func() {
				items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
					Limit:    limit,
					Canaries: []types.ResourceSelector{{Namespace: "logistics"}},
				})

				Expect(err).To(BeNil())
				Expect(limit).To(Equal(len(items.Canaries)))
			})
		}
	})

	ginkgo.Context("It should return the fixed page size for all types", func() {
		for pageSize := 1; pageSize < 3; pageSize++ {
			ginkgo.It(fmt.Sprintf("should work with %d page size", pageSize), func() {
				items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
					Limit:      pageSize,
					Canaries:   []types.ResourceSelector{{Namespace: "logistics"}},
					Configs:    []types.ResourceSelector{{Search: fmt.Sprintf("config_class=%s", models.ConfigClassNode)}},
					Components: []types.ResourceSelector{{Types: []string{"Application"}}},
					Checks:     []types.ResourceSelector{{Types: []string{"http"}, Agent: "all"}},
				})

				Expect(err).To(BeNil())
				Expect(pageSize).To(Equal(len(items.Canaries)))
				Expect(pageSize).To(Equal(len(items.Configs)))
				Expect(pageSize).To(Equal(len(items.Components)))
				Expect(pageSize).To(Equal(len(items.Checks)))
			})
		}
	})
})

var _ = ginkgo.Describe("ResoureSelectorPEG | Sort And Group By", ginkgo.Ordered, func() {
	ginkgo.BeforeAll(func() {
		_ = query.SyncConfigCache(DefaultContext)
	})

	testData := []struct {
		description string
		query       string
		expectedIDs []uuid.UUID
		resource    string
		err         bool
		errMsg      string
	}{
		{
			description: "helm release sort by name",
			query:       `type=Helm::Release @order=-name`,
			expectedIDs: []uuid.UUID{dummy.NginxHelmRelease.ID, dummy.RedisHelmRelease.ID},
			resource:    "config",
		},
		{
			description: "helm release sort by name descending",
			query:       `type=Helm::Release @order=-name`,
			expectedIDs: []uuid.UUID{dummy.RedisHelmRelease.ID, dummy.NginxHelmRelease.ID},
			resource:    "config",
		},
	}

	fmap := map[string]func(context.Context, int, ...types.ResourceSelector) ([]uuid.UUID, error){
		"config":         query.FindConfigIDsByResourceSelector,
		"component":      query.FindComponentIDs,
		"checks":         query.FindCheckIDs,
		"config_changes": query.FindConfigChangeIDsByResourceSelector,
	}

	uuidSliceToString := func(uuids []uuid.UUID) []string {
		return lo.Map(uuids, func(item uuid.UUID, _ int) string { return item.String() })
	}

	ginkgo.Describe("peg search", func() {
		for _, tt := range testData {

			ginkgo.It(tt.description, func() {
				f, ok := fmap[tt.resource]
				Expect(ok).To(BeTrue())
				ids, err := f(DefaultContext, -1, types.ResourceSelector{Search: tt.query})

				if tt.err {
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring(tt.errMsg))
				} else {
					Expect(err).To(BeNil())
					// We convert to strings slice for readable output
					Expect(uuidSliceToString(ids)).To(ConsistOf(uuidSliceToString(tt.expectedIDs)))
				}
			})
		}
	})
})

var _ = ginkgo.Describe("View Resource Selector", func() {
	testData := []struct {
		description       string
		resourceSelectors []types.ResourceSelector
		expectedViews     []models.View
	}{
		{
			description:       "name",
			resourceSelectors: []types.ResourceSelector{{Name: "metrics"}},
			expectedViews:     []models.View{dummy.ImportedDummyViews["mc/metrics"]},
		},
		{
			description:       "namespace + name",
			resourceSelectors: []types.ResourceSelector{{Namespace: dummy.ViewDev.Namespace, Name: dummy.ViewDev.Name}},
			expectedViews:     []models.View{dummy.ViewDev},
		},
		{
			description:       "label selector - single label",
			resourceSelectors: []types.ResourceSelector{{LabelSelector: "environment=production"}},
			expectedViews:     []models.View{dummy.PodView},
		},
		{
			description:       "label selector - multiple labels",
			resourceSelectors: []types.ResourceSelector{{LabelSelector: "team=platform,environment=development"}},
			expectedViews:     []models.View{dummy.ViewDev},
		},
		{
			description:       "namespace with multiple views",
			resourceSelectors: []types.ResourceSelector{{Namespace: "default"}},
			expectedViews:     []models.View{dummy.PodView},
		},
	}

	ginkgo.Describe("FindViewsByResourceSelector", func() {
		for _, test := range testData {
			ginkgo.It(test.description, func() {
				views, err := query.FindViewsByResourceSelector(DefaultContext, -1, test.resourceSelectors...)
				Expect(err).To(BeNil())
				Expect(views).To(HaveLen(len(test.expectedViews)))
				if len(test.expectedViews) > 0 {
					Expect(models.GetIDs(views...)).To(ContainElements(models.GetIDs(test.expectedViews...)))
				}
			})
		}
	})
})

var _ = ginkgo.Describe("Resoure Selector with PEG", ginkgo.Ordered, func() {
	ginkgo.BeforeAll(func() {
		_ = query.SyncConfigCache(DefaultContext)

		// Refresh materialized view for config_summary
		_ = job.RefreshConfigItemSummary7d(DefaultContext)
	})

	// = , != , item in list, item not in list, prefix, suffix, date operations (created_at, updated_at), agent query
	testData := []struct {
		description string
		query       string
		expectedIDs []uuid.UUID
		resource    string
		err         bool
		errMsg      string
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
			query:       `created_at>now-1y`,
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
			expectedIDs: []uuid.UUID{dummy.LogisticsAPIDeployment.ID, dummy.LogisticsAPIReplicaSet.ID},
			resource:    "config",
		},
		{
			description: "config labels query",
			query:       `labels.account=flanksource labels.environment=production`,
			expectedIDs: []uuid.UUID{dummy.EKSCluster.ID, dummy.EC2InstanceB.ID},
			resource:    "config",
		},
		{
			description: "config labels not equal query",
			query:       `labels.account=flanksource labels.environment!=production`,
			expectedIDs: []uuid.UUID{dummy.EC2InstanceA.ID},
			resource:    "config",
		},
		{
			description: "config labels multiple with ,",
			query:       `labels.account=flanksource labels.environment!=production,development`,
			expectedIDs: []uuid.UUID{dummy.EC2InstanceA.ID},
			resource:    "config",
		},
		{
			description: "config array query",
			query:       `config.spec.template.spec.containers[0].name=logistics-api`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPIDeployment.ID, dummy.LogisticsAPIReplicaSet.ID},
			resource:    "config",
		},
		{
			description: "config array query with integer matching",
			query:       `config.spec.template.spec.containers[0].ports[0].containerPort=80`,
			expectedIDs: []uuid.UUID{dummy.LogisticsAPIDeployment.ID, dummy.LogisticsAPIReplicaSet.ID},
			resource:    "config",
		},
		{
			description: "namespace | search | configs",
			query:       "namespace=missioncontrol type=Logistics::DB::RDS",
			expectedIDs: []uuid.UUID{dummy.LogisticsDBRDS.ID},
			resource:    "config",
		},
		{
			description: "name prefix | components",
			query:       "logistics-* type=Application",
			expectedIDs: []uuid.UUID{
				dummy.LogisticsAPI.ID,
				dummy.LogisticsUI.ID,
				dummy.LogisticsWorker.ID,
			},
			resource: "component",
		},
		{
			description: "name prefix | checks",
			query:       "logistics-* type=http",
			expectedIDs: []uuid.UUID{
				dummy.LogisticsAPIHomeHTTPCheck.ID,
				dummy.LogisticsAPIHealthHTTPCheck.ID,
			},
			resource: "checks",
		},
		{
			description: "name prefix | configs",
			query:       "node*",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
				dummy.KubernetesNodeB.ID,
			},
			resource: "config",
		},
		{
			description: "type glob | configs",
			query:       "type=*Deploy* name=logistics*",
			expectedIDs: []uuid.UUID{
				dummy.LogisticsAPIDeployment.ID,
				dummy.LogisticsUIDeployment.ID,
				dummy.LogisticsWorkerDeployment.ID,
			},
			resource: "config",
		},
		{
			description: "tags value search",
			query:       "tags=us-east-1",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
			},
			resource: "config",
		},
		{
			description: "tags value search negate",
			query:       "type=Kubernetes::Node tags!=aws",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeAKSPool1.ID,
			},
			resource: "config",
		},
		{
			description: "tags prefix search",
			query:       "tags=us-*",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
			},
			resource: "config",
		},
		{
			description: "tags suffix search",
			query:       "tags=*east-1",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
			},
			resource: "config",
		},
		{
			description: "labels value search",
			query:       "labels=managed",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config",
		},
		{
			description: "properties unkeyed value search",
			query:       "properties=linux",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config",
		},
		{
			description: "properties unkeyed value search | prefix",
			query:       "properties=us-west*",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config",
		},
		{
			description: "properties unkeyed value search | glob",
			query:       "properties=*west*",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config",
		},
		{
			description: "properties keyed value search",
			query:       "properties.os=linux",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config",
		},
		{
			description: "should throw error for unsupported column",
			query:       "random=column",
			resource:    "config",
			err:         true,
			errMsg:      "not supported",
		},
		{
			description: "config changes by type",
			query:       "change_type=CREATE",
			expectedIDs: []uuid.UUID{
				uuid.MustParse(dummy.EKSClusterCreateChange.ID),
				uuid.MustParse(dummy.KubernetesNodeAChange.ID),
			},
			resource: "config_changes",
		},
		{
			description: "properties unkeyed value search | glob",
			query:       "properties=*west*",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config_summary",
		},
		{
			description: "properties keyed value search",
			query:       "properties.os=linux",
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeB.ID,
			},
			resource: "config_summary",
		},
		{
			description: "config labels not equal query",
			query:       `labels.account=flanksource labels.environment!=production`,
			expectedIDs: []uuid.UUID{dummy.EC2InstanceA.ID},
			resource:    "config_summary",
		},
		{
			description: "config labels multiple with ,",
			query:       `labels.account=flanksource labels.environment!=production,development`,
			expectedIDs: []uuid.UUID{dummy.EC2InstanceA.ID},
			resource:    "config_summary",
		},
		{
			description: "configs with changes",
			query:       `changes>0 type=Helm::Release`,
			expectedIDs: []uuid.UUID{
				dummy.NginxHelmRelease.ID,
				dummy.RedisHelmRelease.ID,
			},
			resource: "config_summary",
		},
		{
			description: "configs with analysis",
			query:       `analysis>0 type=Logistics::DB::RDS`,
			expectedIDs: []uuid.UUID{dummy.LogisticsDBRDS.ID},
			resource:    "config_summary",
		},
		// type defaults to "both" when not specified, so this returns
		// both hard and soft relationships
		{
			description: "related configs | outgoing direction",
			query:       fmt.Sprintf(`related="%s,direction=outgoing"`, dummy.KubernetesCluster.ID.String()),
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
				dummy.KubernetesNodeB.ID,
				dummy.KubernetesNodeAKSPool1.ID,
			},
			resource: "config",
		},
		{
			description: "related configs | incoming direction",
			query:       fmt.Sprintf(`related="%s,direction=incoming"`, dummy.KubernetesNodeA.ID.String()),
			expectedIDs: []uuid.UUID{
				dummy.KubernetesCluster.ID,
			},
			resource: "config",
		},
		{
			description: "related configs | all direction (default)",
			query:       fmt.Sprintf(`related=%s`, dummy.KubernetesNodeA.ID.String()),
			expectedIDs: []uuid.UUID{
				dummy.KubernetesCluster.ID,
			},
			resource: "config",
		},
		{
			description: "related configs | with depth limit",
			query:       fmt.Sprintf(`related="%s,direction=outgoing,depth=1"`, dummy.KubernetesCluster.ID.String()),
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
				dummy.KubernetesNodeB.ID,
				dummy.KubernetesNodeAKSPool1.ID,
			},
			resource: "config",
		},
		{
			description: "related configs | config_summary",
			query:       fmt.Sprintf(`related="%s,direction=outgoing"`, dummy.KubernetesCluster.ID.String()),
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
				dummy.KubernetesNodeB.ID,
				dummy.KubernetesNodeAKSPool1.ID,
			},
			resource: "config_summary",
		},
		{
			description: "related configs | invalid config id",
			query:       `related=invalid-uuid`,
			resource:    "config",
			err:         true,
			errMsg:      "invalid config ID",
		},
		{
			description: "related configs | invalid direction",
			query:       fmt.Sprintf(`related="%s,direction=invalid"`, dummy.KubernetesCluster.ID.String()),
			resource:    "config",
			err:         true,
			errMsg:      "invalid direction",
		},
		{
			description: "related configs | type=hard",
			query:       fmt.Sprintf(`related="%s,direction=outgoing,type=hard"`, dummy.KubernetesCluster.ID.String()),
			expectedIDs: []uuid.UUID{},
			resource:    "config",
		},
		{
			description: "related configs | type=soft",
			query:       fmt.Sprintf(`related="%s,direction=outgoing,type=soft"`, dummy.KubernetesCluster.ID.String()),
			expectedIDs: []uuid.UUID{
				dummy.KubernetesNodeA.ID,
				dummy.KubernetesNodeB.ID,
				dummy.KubernetesNodeAKSPool1.ID,
			},
			resource: "config",
		},
		{
			description: "related configs | invalid type",
			query:       fmt.Sprintf(`related="%s,type=invalid"`, dummy.KubernetesCluster.ID.String()),
			resource:    "config",
			err:         true,
			errMsg:      "invalid type",
		},
	}

	fmap := map[string]func(context.Context, int, ...types.ResourceSelector) ([]uuid.UUID, error){
		"config":         query.FindConfigIDsByResourceSelector,
		"component":      query.FindComponentIDs,
		"checks":         query.FindCheckIDs,
		"config_changes": query.FindConfigChangeIDsByResourceSelector,
		"config_summary": query.FindConfigItemSummaryIDsByResourceSelector,
	}

	uuidSliceToString := func(uuids []uuid.UUID) []string {
		return lo.Map(uuids, func(item uuid.UUID, _ int) string { return item.String() })
	}

	ginkgo.Describe("peg search", func() {
		for _, tt := range testData {
			ginkgo.It(tt.description, func() {
				f, ok := fmap[tt.resource]
				Expect(ok).To(BeTrue())
				ids, err := f(DefaultContext, -1, types.ResourceSelector{Search: tt.query})

				if tt.err {
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring(tt.errMsg))
				} else {
					Expect(err).To(BeNil())
					// We convert to strings slice for readable output
					Expect(uuidSliceToString(ids)).To(ConsistOf(uuidSliceToString(tt.expectedIDs)))
				}
			})
		}
	})
})
