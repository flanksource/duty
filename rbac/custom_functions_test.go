package rbac

import (
	"testing"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
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

func Test_matchPerm(t *testing.T) {
	type args struct {
		obj     models.ABACAttribute
		_agents any
		_tags   string
	}

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "tag only match",
			args: args{
				obj: models.ABACAttribute{
					Config: models.ConfigItem{
						ID: uuid.New(),
						Tags: map[string]string{
							"namespace": "default",
						},
					},
				},
				_agents: "",
				_tags:   "namespace=default",
			},
			want: true,
		},
		{
			name: "multiple tags match",
			args: args{
				obj: models.ABACAttribute{
					Config: models.ConfigItem{
						ID: uuid.New(),
						Tags: map[string]string{
							"namespace": "default",
							"cluster":   "homelab",
						},
					},
				},
				_agents: "",
				_tags:   "namespace=default,cluster=homelab",
			},
			want: true,
		},
		{
			name: "multiple tags no match",
			args: args{
				obj: models.ABACAttribute{
					Config: models.ConfigItem{
						ID: uuid.New(),
						Tags: map[string]string{
							"namespace": "default",
						},
					},
				},
				_agents: "",
				_tags:   "namespace=default,cluster=homelab",
			},
			want: false,
		},
		{
			name: "tags & agents match",
			args: args{
				obj: models.ABACAttribute{
					Config: models.ConfigItem{
						ID: uuid.New(),
						Tags: map[string]string{
							"namespace": "default",
						},
						AgentID: uuid.MustParse("66eda456-315f-455a-95d4-6ef059fc13a8"),
					},
				},
				_agents: "66eda456-315f-455a-95d4-6ef059fc13a8",
				_tags:   "namespace=default",
			},
			want: true,
		},
		{
			name: "tags match & agent no match",
			args: args{
				obj: models.ABACAttribute{
					Config: models.ConfigItem{
						ID: uuid.New(),
						Tags: map[string]string{
							"namespace": "default",
						},
						AgentID: uuid.MustParse("66eda456-315f-455a-95d4-6ef059fc13a8"),
					},
				},
				_agents: "",
				_tags:   "namespace=default,cluster=homelab",
			},
			want: false,
		},
		{
			name: "tags no match & agent match",
			args: args{
				obj: models.ABACAttribute{
					Config: models.ConfigItem{
						ID: uuid.New(),
						Tags: map[string]string{
							"namespace": "default",
						},
						AgentID: uuid.MustParse("66eda456-315f-455a-95d4-6ef059fc13a8"),
					},
				},
				_agents: "66eda456-315f-455a-95d4-6ef059fc13a8",
				_tags:   "namespace=mc",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchPerm(&tt.args.obj, tt.args._agents, tt.args._tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchPerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("matchPerm() = %v, want %v", got, tt.want)
			}
		})
	}
}
