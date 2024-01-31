package types

import (
	"testing"

	"github.com/samber/lo"
)

func TestResourceSelector_Hash_Consistency(t *testing.T) {
	var iteration = 50

	tests := []struct {
		name           string
		resourceSelect ResourceSelector
		expectedHash   string
	}{
		{
			name: "Test Case 1",
			resourceSelect: ResourceSelector{
				Name:          "example",
				Namespace:     "default",
				AgentID:       "123",
				Types:         []string{"a", "b", "c"},
				Statuses:      []string{"healthy", "unhealthy", "terminating"},
				LabelSelector: "app=example,env=production",
				FieldSelector: "owner=admin,path=/,icon=example.png",
			},
			expectedHash: "a7b2305ad03c316162786170090e56ebd0d240b6e1e22c011b4d71b32adb0c4f",
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
