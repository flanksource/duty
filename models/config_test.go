package models

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ptr[T any](v T) *T {
	return &v
}

var _ = Describe("Config", func() {

	It("should remove specified fields", func() {
		id := uuid.New()
		config := ConfigItem{
			ID:   id,
			Name: ptr("dummy-canary"),
		}
		removeFields := []string{"updated_at", "health", "created_at", "config_class", "last_scraped_time"}
		want := map[string]any{
			"name":        "dummy-canary",
			"agent_id":    "00000000-0000-0000-0000-000000000000",
			"ready":       false,
			"description": nil,
			"config":      nil,
			"status":      nil,
			"type":        nil,
			"id":          id.String(),
		}
		got := config.AsMap(removeFields...)
		Expect(got).To(Equal(want))
	})
})
