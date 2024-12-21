package models

import (
	"testing"

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
			expected: `r.obj.playbook.id == "33333333-3333-3333-3333-333333333333"`,
		},
		{
			name: "Multiple fields II",
			perm: Permission{
				ConfigID:   lo.ToPtr(uuid.MustParse("88888888-8888-8888-8888-888888888888")),
				PlaybookID: lo.ToPtr(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
			},
			expected: `r.obj.config.id == "88888888-8888-8888-8888-888888888888" && r.obj.playbook.id == "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`,
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
			expected: `"matchPerm(r.obj, ('aws','azure'), '')"`,
		},
		{
			name: "tags",
			perm: Permission{
				Tags: map[string]string{
					"cluster": "aws",
				},
			},
			expected: `"matchPerm(r.obj, (), 'cluster=aws')"`,
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
