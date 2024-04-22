package models

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T {
	return &v
}

func TestConfig_AsMap(t *testing.T) {
	id := uuid.New()
	fmt.Println(id.String())
	tests := []struct {
		name         string
		config       ConfigItem
		removeFields []string
		want         map[string]any
	}{
		{
			name: "remove single field",
			config: ConfigItem{
				ID:   id,
				Name: ptr("dummy-canary"),
			},
			removeFields: []string{"updated_at", "health", "created_at", "config_class", "last_scraped_time"},
			want: map[string]any{
				"name":     "dummy-canary",
				"agent_id": "00000000-0000-0000-0000-000000000000",
				"ready":    false,
				"id":       id.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.AsMap(tt.removeFields...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("config.AsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
