package models

import (
	"testing"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
)

func TestPermission_Condition(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected string
	}{
		{
			name: "single",
			perm: Permission{
				PlaybookID: lo.ToPtr(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
			},
			expected: `r.obj.playbook != undefined && r.obj.playbook.id == "33333333-3333-3333-3333-333333333333"`,
		},
		{
			name: "Multiple fields II",
			perm: Permission{
				ConfigID:   lo.ToPtr(uuid.MustParse("88888888-8888-8888-8888-888888888888")),
				PlaybookID: lo.ToPtr(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
			},
			expected: `r.obj.config != undefined && r.obj.config.id == "88888888-8888-8888-8888-888888888888" && r.obj.playbook != undefined && r.obj.playbook.id == "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`,
		},
		{
			name:     "No fields set",
			perm:     Permission{},
			expected: "",
		},
		{
			name: "agents",
			perm: Permission{
				Agents: pq.StringArray([]string{"aws", "azure"}),
			},
			expected: "r.obj.config != undefined && r.obj.config.agent_id in ('aws','azure') && r.obj.component != undefined && r.obj.component.agent_id in ('aws','azure') && r.obj.canary != undefined && r.obj.canary.agent_id in ('aws','azure')",
		},
		{
			name: "tags",
			perm: Permission{
				Tags: types.JSONStringMap{
					"cluster": "aws",
				},
			},
			expected: `r.obj.config != undefined && mapContains("{\"cluster\":\"aws\"}", r.obj.config.tags)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perm.Condition()
			if tt.expected != result {
				t.Errorf("Expected %s\nGot %s", tt.expected, result)
			}
		})
	}
}
