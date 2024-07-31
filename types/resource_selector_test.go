package types_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

func TestResourceSelector_Hash_Consistency(t *testing.T) {
	var iteration = 50

	tests := []struct {
		name           string
		resourceSelect types.ResourceSelector
		expectedHash   string
	}{
		{
			name: "Test Case 1",
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
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < iteration; i++ {
				tt.resourceSelect.Types = lo.Shuffle(tt.resourceSelect.Types)
				tt.resourceSelect.Statuses = lo.Shuffle(tt.resourceSelect.Statuses)

				actualHash := tt.resourceSelect.Hash()
				if tt.expectedHash != actualHash {
					t.Errorf("[%s] Hash mismatch. expected(%s) got(%s)", tt.name, tt.expectedHash, actualHash)
				}
			}
		})
	}
}

func TestResourceSelector_Matches(t *testing.T) {
	tests := []struct {
		name             string
		resourceSelector types.ResourceSelector
		selectable       types.ResourceSelectable
		unselectable     types.ResourceSelectable
	}{
		{
			name:             "Blank",
			resourceSelector: types.ResourceSelector{},
			selectable:       nil,
			unselectable: models.ConfigItem{
				Name: lo.ToPtr("silverbullet"),
				Labels: &types.JSONStringMap{
					"namespace": "default",
				},
			},
		},
		{
			name: "ID",
			resourceSelector: types.ResourceSelector{
				ID: "4775d837-727a-4386-9225-1fa2c167cc96",
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
			resourceSelector: types.ResourceSelector{
				Name:      "airsonic",
				Namespace: "default",
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
			name: "Types",
			resourceSelector: types.ResourceSelector{
				Types: []string{"Kubernetes::Pod"},
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
			resourceSelector: types.ResourceSelector{
				Namespace: "default",
				Statuses:  []string{"healthy"},
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
			name: "Types",
			resourceSelector: types.ResourceSelector{
				Types: []string{"Kubernetes::Pod"},
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
			name: "Label selector",
			resourceSelector: types.ResourceSelector{
				Namespace:     "default",
				LabelSelector: "env=production",
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
			resourceSelector: types.ResourceSelector{
				Namespace:     "default",
				LabelSelector: "env in (production)",
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
			name: "Field selector",
			resourceSelector: types.ResourceSelector{
				Namespace:     "default",
				FieldSelector: "config_class=Cluster",
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
			resourceSelector: types.ResourceSelector{
				Namespace:     "default",
				FieldSelector: "config_class notin (Cluster)",
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
			resourceSelector: types.ResourceSelector{
				FieldSelector: "color=red",
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
			name: "Field selector property matcher (value)",
			resourceSelector: types.ResourceSelector{
				FieldSelector: "memory>50",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.selectable != nil && !tt.resourceSelector.Matches(tt.selectable) {
				t.Errorf("failed[%s] expected to match", tt.name)
			}

			if tt.unselectable != nil && tt.resourceSelector.Matches(tt.unselectable) {
				t.Errorf("expected to not match")
			}
		})
	}
}
