package models

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestCanary_AsMap(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name         string
		canary       Canary
		removeFields []string
		want         map[string]any
	}{
		{
			name: "remove single field",
			canary: Canary{
				ID:        id,
				Namespace: "canary",
				Name:      "dummy-canary",
			},
			want: map[string]any{
				"name":       "dummy-canary",
				"namespace":  "canary",
				"agent_id":   "00000000-0000-0000-0000-000000000000",
				"created_at": "0001-01-01T00:00:00Z",
				"updated_at": "0001-01-01T00:00:00Z",
				"id":         id.String(),
				"spec":       nil,
			},
		},
		{
			name: "remove multiple fields",
			canary: Canary{
				ID:        uuid.New(),
				Namespace: "canary",
				Name:      "dummy-canary",
			},
			removeFields: []string{"id", "created_at", "agent_id", "updated_at"},
			want: map[string]any{
				"name":      "dummy-canary",
				"namespace": "canary",
				"spec":      nil,
			},
		},
		{
			name: "remove no fields",
			canary: Canary{
				Namespace: "canary",
				Name:      "dummy-canary",
			},
			removeFields: nil,
			want: map[string]any{
				"name":       "dummy-canary",
				"namespace":  "canary",
				"id":         "00000000-0000-0000-0000-000000000000",
				"agent_id":   "00000000-0000-0000-0000-000000000000",
				"created_at": "0001-01-01T00:00:00Z",
				"updated_at": "0001-01-01T00:00:00Z",
				"spec":       nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.canary.AsMap(tt.removeFields...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Canary.AsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
