package postgrest_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/schema/openapi/postgrest"
)

func TestPostgrest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgrest OpenAPI Suite")
}

const minimalSwagger = `{
  "swagger": "2.0",
  "info": { "title": "PostgREST API", "version": "11.0.0" },
  "host": "localhost:3000",
  "basePath": "/",
  "schemes": ["http"],
  "consumes": ["application/json"],
  "produces": ["application/json"],
  "paths": {
    "/config_items": {
      "get": {
        "tags": ["config_items"],
        "responses": {
          "200": {
            "description": "OK",
            "schema": { "type": "array", "items": { "$ref": "#/definitions/config_items" } }
          }
        }
      }
    },
    "/rpc/example": {
      "post": {
        "responses": { "200": { "description": "OK", "schema": { "type": "object" } } }
      }
    }
  },
  "definitions": {
    "config_items": {
      "type": "object",
      "properties": {
        "id":   { "type": "string", "format": "uuid" },
        "name": { "type": "string" },
        "type": { "type": "string" },
        "tags": { "type": "object" }
      }
    }
  }
}`

var _ = Describe("Convert + Enrich + Tag pipeline", func() {
	It("converts Swagger 2.0 to a valid OpenAPI 3.1 document", func() {
		spec, err := postgrest.Convert([]byte(minimalSwagger))
		Expect(err).ToNot(HaveOccurred())
		Expect(spec.OpenAPI).To(Equal("3.1.0"))
		Expect(spec.Paths.Find("/config_items")).ToNot(BeNil())
		Expect(spec.Components.Schemas).To(HaveKey("config_items"))
	})

	It("runs enrichment without error and reports registry hits/misses", func() {
		spec, err := postgrest.Convert([]byte(minimalSwagger))
		Expect(err).ToNot(HaveOccurred())

		res, err := postgrest.Enrich(spec)
		Expect(err).ToNot(HaveOccurred())

		// config_items is registered, so the schema is recognised. Whether
		// it gains descriptions depends on whether the Go struct fields
		// carry doc comments; we only assert the structural behaviour.
		Expect(res.UnknownTables).ToNot(ContainElement("config_items"))
	})

	It("flags spec schemas that have no Register(...) entry as unknown", func() {
		swagger := `{
		  "swagger": "2.0",
		  "info": {"title": "x", "version": "0"},
		  "paths": {},
		  "definitions": {
		    "made_up_table_xyz": {
		      "type": "object",
		      "properties": {"id": {"type": "string"}}
		    }
		  }
		}`
		spec, err := postgrest.Convert([]byte(swagger))
		Expect(err).ToNot(HaveOccurred())

		res, err := postgrest.Enrich(spec)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.UnknownTables).To(ContainElement("made_up_table_xyz"))
	})

	It("tags every operation that maps to a known RBAC object", func() {
		spec, err := postgrest.Convert([]byte(minimalSwagger))
		Expect(err).ToNot(HaveOccurred())
		postgrest.TagWithRBAC(spec)

		op := spec.Paths.Find("/config_items").Get
		Expect(op).ToNot(BeNil())
		Expect(op.Extensions).To(HaveKeyWithValue("x-rbac-object", "catalog"))
		Expect(op.Extensions).To(HaveKeyWithValue("x-rbac-action", "read"))
		Expect(op.Tags).To(ContainElement("catalog"))

		tagNames := make([]string, 0, len(spec.Tags))
		for _, t := range spec.Tags {
			tagNames = append(tagNames, t.Name)
		}
		Expect(tagNames).To(ContainElement("catalog"))
	})

	It("applies an RPC overlay to a /rpc/<name> POST 200 response", func() {
		spec, err := postgrest.Convert([]byte(minimalSwagger))
		Expect(err).ToNot(HaveOccurred())

		overlay := &postgrest.RPCOverlay{
			RPCs: map[string]struct {
				Response map[string]any `json:"response,omitempty" yaml:"response,omitempty"`
			}{
				"example": {
					Response: map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id": map[string]any{"type": "string", "format": "uuid"},
							},
						},
					},
				},
			},
		}

		applied, err := postgrest.ApplyRPCOverlay(spec, overlay)
		Expect(err).ToNot(HaveOccurred())
		Expect(applied).To(Equal(1))

		mt := spec.Paths.Find("/rpc/example").Post.Responses.Status(200).Value.Content.Get("application/json")
		Expect(mt).ToNot(BeNil())
		Expect(mt.Schema.Value.Type.Is("array")).To(BeTrue())
	})
})

var _ = Describe("Registry", func() {
	It("contains the high-traffic tables", func() {
		for _, name := range []string{"config_items", "components", "checks", "playbooks", "notifications"} {
			r, ok := postgrest.Lookup(name)
			Expect(ok).To(BeTrue(), "expected %s to be registered", name)
			Expect(r.GoType).ToNot(BeNil())
			Expect(r.Kind).To(Equal(postgrest.KindTable))
		}
	})
})
