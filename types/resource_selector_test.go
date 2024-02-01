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
			expectedHash: "96db782c434227b234c636aa9bfac70f1590146414dfd04263b4dc38c2f13444",
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
