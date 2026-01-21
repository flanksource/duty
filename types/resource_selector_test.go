package types_test

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/samber/lo/mutable"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var _ = Describe("Resource Selector", func() {
	iteration := 50

	tests := []struct {
		name           string
		resourceSelect types.ResourceSelector
		expectedHash   string
	}{
		{
			name: "Hash Equality",
			resourceSelect: types.ResourceSelector{
				ID:            "4775d837-727a-4386-9225-1fa2c167cc96",
				Name:          "example",
				Namespace:     "default",
				Agent:         "123",
				Types:         []string{"a", "b", "c"},
				Statuses:      []string{"healthy", "unhealthy", "terminating"},
				LabelSelector: "app=example,env=production",
				FieldSelector: "owner=admin,path=/,icon=example.png",
			},
			expectedHash: "f591f8377f280e4e8a29695d70ab237e2862c9d594f073cfb145a8c55f709a0e",
		},
	}

	for _, tt := range tests {
		It(tt.name, func() {
			for i := 0; i < iteration; i++ {
				mutable.Shuffle(tt.resourceSelect.Types)
				mutable.Shuffle(tt.resourceSelect.Statuses)

				actualHash := tt.resourceSelect.Hash()
				Expect(actualHash).To(Equal(tt.resourceSelect.Hash()))
			}
		})
	}

	Describe("Matches", func() {
		tests := []struct {
			name              string
			resourceSelectors []types.ResourceSelector // canonical resource selectors
			selectable        types.ResourceSelectable
			unselectable      types.ResourceSelectable
		}{
			{
				name:              "Blank",
				resourceSelectors: []types.ResourceSelector{},
				selectable:        nil,
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Labels: &types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "ID",
				resourceSelectors: []types.ResourceSelector{
					{ID: "4775d837-727a-4386-9225-1fa2c167cc96"},
				},
				selectable: models.ConfigItem{
					ID:   uuid.MustParse("4775d837-727a-4386-9225-1fa2c167cc96"),
					Name: lo.ToPtr("silverbullet"),
				},
				unselectable: models.ConfigItem{
					ID:   uuid.MustParse("5775d837-727a-4386-9225-1fa2c167cc96"),
					Name: lo.ToPtr("silverbullet"),
				},
			},
			{
				name: "Namespace & Name",
				resourceSelectors: []types.ResourceSelector{
					{Name: "airsonic", Namespace: "default"},
					{Search: "name=airsonic namespace=default"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("airsonic"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "Namespace wildcard ignored",
				resourceSelectors: []types.ResourceSelector{
					{Name: "airsonic", Namespace: "*"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("airsonic"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "Tag selector wildcard exists",
				resourceSelectors: []types.ResourceSelector{
					{TagSelector: "cluster=*"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Tags: types.JSONStringMap{
						"cluster": "prod",
					},
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "Tag selector wildcard with requirement",
				resourceSelectors: []types.ResourceSelector{
					{TagSelector: "cluster=*,namespace=default"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Tags: types.JSONStringMap{
						"cluster":   "prod",
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "Label selector wildcard exists",
				resourceSelectors: []types.ResourceSelector{
					{LabelSelector: "app=*,tier=backend"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Labels: &types.JSONStringMap{
						"app":  "api",
						"tier": "backend",
					},
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
					Labels: &types.JSONStringMap{
						"tier": "backend",
					},
				},
			},
			{
				name: "Field selector wildcard ignored",
				resourceSelectors: []types.ResourceSelector{
					{Name: "airsonic", FieldSelector: "owner=*"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("airsonic"),
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("silverbullet"),
				},
			},
			{
				name: "Types",
				resourceSelectors: []types.ResourceSelector{
					{Types: []string{"Kubernetes::Pod"}},
					{Search: "type=Kubernetes::Pod"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("cert-manager"),
					Type: lo.ToPtr("Kubernetes::Pod"),
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("cert-manager"),
					Type: lo.ToPtr("Kubernetes::Deployment"),
				},
			},
			{
				name: "Types multiple",
				resourceSelectors: []types.ResourceSelector{
					{Types: []string{"Kubernetes::Node", "Kubernetes::Pod"}},
					{Search: "type=Kubernetes::Node,Kubernetes::Pod"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("cert-manager"),
					Type: lo.ToPtr("Kubernetes::Pod"),
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("cert-manager"),
					Type: lo.ToPtr("Kubernetes::Deployment"),
				},
			},
			{
				name: "Type negatives",
				resourceSelectors: []types.ResourceSelector{
					{Types: []string{"!Kubernetes::Deployment", "Kubernetes::Pod"}},
					{Search: "type=Kubernetes::Pod type!=Kubernetes::Deployment"},
				},
				selectable: models.ConfigItem{
					Name: lo.ToPtr("cert-manager"),
					Type: lo.ToPtr("Kubernetes::Pod"),
				},
				unselectable: models.ConfigItem{
					Name: lo.ToPtr("cert-manager"),
					Type: lo.ToPtr("Kubernetes::Deployment"),
				},
			},
			{
				name: "Statuses",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", Statuses: []string{"healthy"}},
					{Search: "namespace=default status=healthy"},
				},
				selectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					Status: lo.ToPtr("healthy"),
				},
				unselectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					Status: lo.ToPtr("unhealthy"),
				},
			},
			{
				name: "Healths",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", Health: "healthy"},
					{Search: "namespace=default health=healthy"},
				},
				selectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					Health: lo.ToPtr(models.HealthHealthy),
				},
				unselectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					Health: lo.ToPtr(models.HealthUnhealthy),
				},
			},
			{
				name: "Healths multiple",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", Health: "healthy,warning"},
					{Search: "namespace=default health=healthy,warning"},
				},
				selectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					Health: lo.ToPtr(models.HealthHealthy),
				},
				unselectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					Health: lo.ToPtr(models.HealthUnhealthy),
				},
			},
			{
				name: "Label selector",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", LabelSelector: "env=production"},
					{Search: "namespace=default labels.env=production"},
				},
				selectable: models.ConfigItem{
					ConfigClass: "Cluster",
					Labels: lo.ToPtr(types.JSONStringMap{
						"env": "production",
					}),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					ConfigClass: "Cluster",
					Labels: lo.ToPtr(types.JSONStringMap{
						"env": "dev",
					}),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "Label selector IN query",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", LabelSelector: "env in (production)"},
				},
				selectable: models.ConfigItem{
					ConfigClass: "Cluster",
					Labels: lo.ToPtr(types.JSONStringMap{
						"env": "production",
					}),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					ConfigClass: "Cluster",
					Labels: lo.ToPtr(types.JSONStringMap{
						"env": "dev",
					}),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
			{
				name: "Label selector EXISTS query",
				resourceSelectors: []types.ResourceSelector{
					{Types: []string{"AWS::AvailabilityZone"}, LabelSelector: "account"},
				},
				selectable: models.ConfigItem{
					Type: lo.ToPtr("AWS::AvailabilityZone"),
					Labels: lo.ToPtr(types.JSONStringMap{
						"account": "prod-account",
						"region":  "us-east-1",
					}),
				},
				unselectable: models.ConfigItem{
					Type: lo.ToPtr("AWS::AvailabilityZone"),
					Labels: lo.ToPtr(types.JSONStringMap{
						"region": "us-east-1",
					}),
				},
			},
			{
				name: "Label selector DOES NOT EXIST query",
				resourceSelectors: []types.ResourceSelector{
					{Types: []string{"AWS::AvailabilityZone"}, LabelSelector: "!account"},
				},
				selectable: models.ConfigItem{
					Type: lo.ToPtr("AWS::AvailabilityZone"),
					Labels: lo.ToPtr(types.JSONStringMap{
						"region": "us-east-1",
					}),
				},
				unselectable: models.ConfigItem{
					Type: lo.ToPtr("AWS::AvailabilityZone"),
					Labels: lo.ToPtr(types.JSONStringMap{
						"account": "prod-account",
						"region":  "us-east-1",
					}),
				},
			},
			{
				name: "Tag selector",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", TagSelector: "cluster=aws"},
					{Search: "namespace=default tags.cluster=aws"},
				},
				selectable: models.ConfigItem{
					ConfigClass: "Cluster",
					Tags: types.JSONStringMap{
						"cluster":   "aws",
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					ConfigClass: "Cluster",
					Tags: types.JSONStringMap{
						"cluster":   "workload",
						"namespace": "default",
					},
				},
			},
			{
				name: "Field selector",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", FieldSelector: "config_class=Cluster"},
				},
				selectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					ConfigClass: "Cluster",
				},
				unselectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					ConfigClass: "VirtualMachine",
				},
			},
			{
				name: "Field selector NOT IN query",
				resourceSelectors: []types.ResourceSelector{
					{Namespace: "default", FieldSelector: "config_class notin (Cluster)"},
				},
				selectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					ConfigClass: "VirtualMachine",
				},
				unselectable: models.ConfigItem{
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
					ConfigClass: "Cluster",
				},
			},
			{
				name: "Field selector property matcher (text)",
				resourceSelectors: []types.ResourceSelector{
					{FieldSelector: "properties.color=red"},
				},
				selectable: models.ConfigItem{
					Properties: &types.Properties{
						{Name: "color", Text: "red"},
					},
				},
				unselectable: models.ConfigItem{
					Properties: &types.Properties{
						{Name: "color", Text: "green"},
					},
				},
			},
			{
				name: "Property selector",
				resourceSelectors: []types.ResourceSelector{
					{FieldSelector: "properties.memory>50"},
				},
				selectable: models.ConfigItem{
					Properties: &types.Properties{
						{Name: "memory", Value: lo.ToPtr(int64(64))},
					},
				},
				unselectable: models.ConfigItem{
					Properties: &types.Properties{
						{Name: "memory", Value: lo.ToPtr(int64(32))},
					},
				},
			},
			{
				name: "Selectable Map",
				resourceSelectors: []types.ResourceSelector{
					{Name: "airsonic", Namespace: "music", Types: []string{"Kubernetes::Deployment"}},
				},
				selectable: types.ResourceSelectableMap{
					"name": "airsonic",
					"type": "Kubernetes::Deployment",
					"tags": map[string]string{
						"namespace": "music",
					},
				},
				unselectable: types.ResourceSelectableMap{
					"name": "airsonic",
					"type": "Kubernetes::Pod",
					"tags": map[string]string{
						"namespace": "music",
					},
				},
			},
			{
				name: "Agent - matches ConfigItem with same agent",
				resourceSelectors: []types.ResourceSelector{
					{Agent: "ac4b1dc5-b249-471d-89d7-ba0c5de4997b"},
					{Search: "agent=ac4b1dc5-b249-471d-89d7-ba0c5de4997b"},
				},
				selectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.MustParse("ac4b1dc5-b249-471d-89d7-ba0c5de4997b"),
					Name:    lo.ToPtr("homelab-vm"),
				},
				unselectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.MustParse("12345678-1234-1234-1234-123456789012"),
					Name:    lo.ToPtr("gcp-vm"),
				},
			},
			{
				name: "Agent - does not match ConfigItem with no agent",
				resourceSelectors: []types.ResourceSelector{
					{Agent: "ac4b1dc5-b249-471d-89d7-ba0c5de4997b"},
				},
				selectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.MustParse("ac4b1dc5-b249-471d-89d7-ba0c5de4997b"),
					Name:    lo.ToPtr("homelab-vm"),
				},
				unselectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.Nil,
					Name:    lo.ToPtr("central-db"),
				},
			},
			{
				name: "Agent with multiple criteria",
				resourceSelectors: []types.ResourceSelector{
					{Agent: "ac4b1dc5-b249-471d-89d7-ba0c5de4997b", Namespace: "default"},
				},
				selectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.MustParse("ac4b1dc5-b249-471d-89d7-ba0c5de4997b"),
					Name:    lo.ToPtr("homelab-vm"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.MustParse("ac4b1dc5-b249-471d-89d7-ba0c5de4997b"),
					Name:    lo.ToPtr("homelab-vm-kube-system"),
					Tags: types.JSONStringMap{
						"namespace": "kube-system",
					},
				},
			},
			{
				name: "Agent with multiple criteria - II",
				resourceSelectors: []types.ResourceSelector{
					{Agent: "ac4b1dc5-b249-471d-89d7-ba0c5de4997b", Namespace: "default"},
				},
				selectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.MustParse("ac4b1dc5-b249-471d-89d7-ba0c5de4997b"),
					Name:    lo.ToPtr("homelab-vm"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
				unselectable: models.ConfigItem{
					ID:      uuid.New(),
					AgentID: uuid.New(),
					Name:    lo.ToPtr("homelab-vm-kube-system"),
					Tags: types.JSONStringMap{
						"namespace": "default",
					},
				},
			},
		}

		Describe("test", func() {
			for _, tt := range tests {
				// if tt.name != "Field selector" {
				// 	continue
				// }

				It(tt.name, func() {
					if tt.selectable != nil {
						for _, rs := range tt.resourceSelectors {
							Expect(rs.Matches(tt.selectable)).To(BeTrue(), fmt.Sprintf("%v", rs))
						}
					}

					if tt.unselectable != nil {
						for _, rs := range tt.resourceSelectors {
							Expect(rs.Matches(tt.unselectable)).To(BeFalse(), fmt.Sprintf("%v", rs))
						}
					}
				})
			}
		})

		Describe("Canonical", func() {
			It("should normalize wildcard values", func() {
				rs := types.ResourceSelector{
					ID:            "*",
					Name:          "*",
					Namespace:     "*",
					Agent:         "*",
					Scope:         "*",
					TagSelector:   "cluster=*",
					LabelSelector: "app=*,tier=backend",
					FieldSelector: "owner=*",
					Types:         []string{"*", "Kubernetes::Pod"},
					Statuses:      []string{"*", "healthy"},
					Health:        "*,warning",
				}

				canonical := rs.Canonical()
				Expect(canonical.ID).To(Equal(""))
				Expect(canonical.Name).To(Equal("*"))
				Expect(canonical.Namespace).To(Equal(""))
				Expect(canonical.Agent).To(Equal("all"))
				Expect(canonical.Scope).To(Equal(""))
				Expect(canonical.TagSelector).To(Equal("cluster"))
				Expect(canonical.LabelSelector).To(Equal("app,tier=backend"))
				Expect(canonical.FieldSelector).To(Equal(""))
				Expect(canonical.Types).To(Equal(types.Items{"Kubernetes::Pod"}))
				Expect(canonical.Statuses).To(Equal(types.Items{"healthy"}))
				Expect(canonical.Health).To(Equal(types.MatchExpression("warning")))
			})
		})
	})
})
