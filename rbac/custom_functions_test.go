package rbac

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

func Test_matchResourceSelector(t *testing.T) {
	type args struct {
		attr     models.ABACAttribute
		selector Selectors
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "basic namespace/name match",
			want: true,
			args: args{
				attr: models.ABACAttribute{
					Config: models.ConfigItem{
						ID:   uuid.New(),
						Name: lo.ToPtr("airsonic"),
						Tags: map[string]string{
							"namespace": "default",
						},
					},
				},
				selector: Selectors{
					Configs: []types.ResourceSelector{
						{
							Namespace: "default",
							Name:      "airsonic",
						},
					},
				},
			},
		},
		{
			name: "1 attribute, 1 selector, no match",
			want: false,
			args: args{
				attr: models.ABACAttribute{
					Config: models.ConfigItem{
						ID:   uuid.New(),
						Name: lo.ToPtr("airsonic"),
						Tags: map[string]string{
							"namespace": "default",
						},
					},
				},
				selector: Selectors{
					Playbooks: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
				},
			},
		},
		{
			name: "1 attribute, 1 selector, match",
			want: true,
			args: args{
				attr: models.ABACAttribute{
					Config: models.ConfigItem{
						ID:   uuid.New(),
						Name: lo.ToPtr("airsonic"),
						Tags: map[string]string{
							"namespace": "default",
						},
					},
				},
				selector: Selectors{
					Configs: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
				},
			},
		},
		{
			name: "2 attributes, 2 selectors, no match",
			want: false,
			args: args{
				attr: models.ABACAttribute{
					Connection: models.Connection{
						ID:   uuid.New(),
						Name: "gemini",
					},
					Playbook: models.Playbook{
						ID:   uuid.New(),
						Name: "diagnose-airsonic",
					},
				},
				selector: Selectors{
					Playbooks: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
					Configs: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
				},
			},
		},
		{
			name: "2 attributes, 2 selectors, match",
			want: true,
			args: args{
				attr: models.ABACAttribute{
					Connection: models.Connection{
						ID:   uuid.New(),
						Name: "gemini",
					},
					Playbook: models.Playbook{
						ID:   uuid.New(),
						Name: "diagnose-airsonic",
					},
				},
				selector: Selectors{
					Playbooks: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
					Connections: []types.ResourceSelector{
						{
							Name: "gemini",
						},
					},
				},
			},
		},
		{
			name: "2 attributes, 1 selector, match",
			want: true,
			args: args{
				attr: models.ABACAttribute{
					Connection: models.Connection{
						ID:   uuid.New(),
						Name: "gemini",
					},
					Playbook: models.Playbook{
						ID:   uuid.New(),
						Name: "diagnose-airsonic",
					},
				},
				selector: Selectors{
					Playbooks: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
				},
			},
		},
		{
			name: "1 attribute, 2 selectors, no match",
			want: false,
			args: args{
				attr: models.ABACAttribute{
					Connection: models.Connection{
						ID:   uuid.New(),
						Name: "gemini",
					},
				},
				selector: Selectors{
					Playbooks: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
					Connections: []types.ResourceSelector{
						{
							Name: "*",
						},
					},
				},
			},
		},
		{
			name: "1 attribute, 2 selectors for same resource, match",
			want: true,
			args: args{
				attr: models.ABACAttribute{
					Connection: models.Connection{
						ID:   uuid.New(),
						Name: "gemini",
					},
				},
				selector: Selectors{
					Connections: []types.ResourceSelector{
						{Name: "anthropic"},
						{Name: "gemini"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchResourceSelector(&tt.args.attr, tt.args.selector)
			if err != nil {
				t.Errorf("matchResourceSelector() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("matchResourceSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
