package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

func TestPermission_Principal(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected string
	}{
		{
			name: "PersonID only",
			perm: Permission{
				PersonID: lo.ToPtr(uuid.MustParse("11111111-1111-1111-1111-111111111111")),
			},
			expected: "r.sub.id == 11111111-1111-1111-1111-111111111111",
		},
		{
			name: "TeamID only",
			perm: Permission{
				TeamID: lo.ToPtr(uuid.MustParse("22222222-2222-2222-2222-222222222222")),
			},
			expected: "r.sub.id == 22222222-2222-2222-2222-222222222222",
		},
		{
			name: "Multiple fields",
			perm: Permission{
				PersonID: lo.ToPtr(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
				ConfigID: lo.ToPtr(uuid.MustParse("55555555-5555-5555-5555-555555555555")),
			},
			expected: "r.sub.id == 33333333-3333-3333-3333-333333333333 && r.config.id == 55555555-5555-5555-5555-555555555555",
		},
		{
			name: "Multiple fields II",
			perm: Permission{
				PersonID:   lo.ToPtr(uuid.MustParse("66666666-6666-6666-6666-666666666666")),
				ConfigID:   lo.ToPtr(uuid.MustParse("88888888-8888-8888-8888-888888888888")),
				PlaybookID: lo.ToPtr(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
			},
			expected: "r.sub.id == 66666666-6666-6666-6666-666666666666 && r.config.id == 88888888-8888-8888-8888-888888888888 && r.playbook.id == aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
		{
			name:     "No fields set",
			perm:     Permission{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perm.Principal()
			if tt.expected != result {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
