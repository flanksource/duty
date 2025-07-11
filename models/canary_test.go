package models

import (
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Canary", func() {
	var (
		id uuid.UUID
	)

	ginkgo.BeforeEach(func() {
		id = uuid.New()
	})

	ginkgo.Describe("AsMap", func() {
		ginkgo.It("should remove single field", func() {
			canary := Canary{
				ID:        id,
				Namespace: "canary",
				Name:      "dummy-canary",
			}
			expected := map[string]any{
				"name":       "dummy-canary",
				"namespace":  "canary",
				"agent_id":   "00000000-0000-0000-0000-000000000000",
				"created_at": "0001-01-01T00:00:00Z",
				"updated_at": nil,
				"id":         id.String(),
				"spec":       nil,
			}
			Expect(canary.AsMap()).To(Equal(expected))
		})

		ginkgo.It("should remove multiple fields", func() {
			canary := Canary{
				ID:        uuid.New(),
				Namespace: "canary",
				Name:      "dummy-canary",
			}
			removeFields := []string{"id", "created_at", "agent_id", "updated_at"}
			expected := map[string]any{
				"name":      "dummy-canary",
				"namespace": "canary",
				"spec":      nil,
			}
			Expect(canary.AsMap(removeFields...)).To(Equal(expected))
		})

		ginkgo.It("should remove no fields", func() {
			canary := Canary{
				Namespace: "canary",
				Name:      "dummy-canary",
			}
			expected := map[string]any{
				"name":       "dummy-canary",
				"namespace":  "canary",
				"id":         "00000000-0000-0000-0000-000000000000",
				"agent_id":   "00000000-0000-0000-0000-000000000000",
				"created_at": "0001-01-01T00:00:00Z",
				"updated_at": nil,
				"spec":       nil,
			}
			Expect(canary.AsMap()).To(Equal(expected))
		})
	})
})
