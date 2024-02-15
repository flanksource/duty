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
				ID:            "4775d837-727a-4386-9225-1fa2c167cc96",
				Name:          "example",
				Namespace:     "default",
				Agent:         "123",
				Types:         []string{"a", "b", "c"},
				Statuses:      []string{"healthy", "unhealthy", "terminating"},
				LabelSelector: "app=example,env=production",
				FieldSelector: "owner=admin,path=/,icon=example.png",
			},
			expectedHash: "56dc1d9aee98f3fad1334fd387e30aa59ce7857f802413a240c60d4724991bf1",
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
