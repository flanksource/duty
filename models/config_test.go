package models

import (
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ptr[T any](v T) *T {
	return &v
}

var _ = ginkgo.Describe("AsMap", func() {
	ginkgo.Context("ConfigItem", func() {
		ginkgo.It("should remove specified fields", func() {
			id := uuid.New()
			config := ConfigItem{
				ID:   id,
				Name: ptr("dummy-canary"),
				Config: ptr(`{
					"name": "dummy-canary",
					"agent_id": "00000000-0000-0000-0000-000000000000",
					"ready": false,
					"description": null,
					"config": null
				}`),
			}

			removeFields := []string{"updated_at", "health", "created_at", "config_class", "last_scraped_time"}
			want := map[string]any{
				"name":        "dummy-canary",
				"agent_id":    "00000000-0000-0000-0000-000000000000",
				"ready":       false,
				"description": nil,
				"tags":        map[string]string{},
				"labels":      map[string]string{},
				"config": map[string]any{
					"name":        "dummy-canary",
					"agent_id":    "00000000-0000-0000-0000-000000000000",
					"ready":       false,
					"description": nil,
					"config":      nil,
				},
				"status": nil,
				"type":   nil,
				"id":     id.String(),
			}
			got := config.AsMap(removeFields...)
			Expect(got).To(Equal(want))
		})
	})

	ginkgo.Context("CatalogChange", func() {
		ginkgo.It("should return details as a map", func() {
			change := CatalogChange{
				ID:         uuid.New(),
				ConfigID:   uuid.New(),
				ChangeType: "UPDATE",
				Summary:    ptr("Helm chart upgraded from 18.1.3 to 18.1.5"),
				Details: []byte(`{
		"old_version": "18.1.3",
		"new_version": "18.1.5"
	}`),
				Count: 1,
			}

			got := change.AsMap()
			Expect(got["details"]).To(Equal(map[string]any{
				"old_version": "18.1.3",
				"new_version": "18.1.5",
			}))
		})
	})
})
