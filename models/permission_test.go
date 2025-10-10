package models

import (
	"testing"

	"github.com/google/uuid"
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
			expected: `str(r.obj.Playbook.ID) == "33333333-3333-3333-3333-333333333333"`,
		},
		{
			name: "Multiple fields II",
			perm: Permission{
				ConfigID:   lo.ToPtr(uuid.MustParse("88888888-8888-8888-8888-888888888888")),
				PlaybookID: lo.ToPtr(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
			},
			expected: `str(r.obj.Config.ID) == "88888888-8888-8888-8888-888888888888" && str(r.obj.Playbook.ID) == "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`,
		},
		{
			name:     "No fields set",
			perm:     Permission{},
			expected: "",
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
