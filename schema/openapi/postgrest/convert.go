package postgrest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

// Convert turns the Swagger 2.0 JSON document emitted by PostgREST into a
// validated OpenAPI 3.1 document. The output is the parsed *openapi3.T so
// callers can keep mutating it through the enrichment pipeline before
// serialising once at the end.
//
// PostgREST emits raw SQL expression strings as column defaults
// (e.g. "generate_ulid()", "now()", "format_incident_id(NEXTVAL(...))"),
// which the OpenAPI 3.1 strict validator then rejects against the column's
// own type constraints (maxLength, format, etc.). Those defaults are not
// useful to API consumers anyway — the server fills them in — so we drop
// every property default before validation.
func Convert(swaggerJSON []byte) (*openapi3.T, error) {
	var v2 openapi2.T
	if err := json.Unmarshal(swaggerJSON, &v2); err != nil {
		return nil, fmt.Errorf("parse swagger 2.0: %w", err)
	}

	v3, err := openapi2conv.ToV3(&v2)
	if err != nil {
		return nil, fmt.Errorf("convert swagger 2.0 to openapi 3: %w", err)
	}

	v3.OpenAPI = "3.1.0"
	if v3.Info != nil && v3.Info.Title == "" {
		v3.Info.Title = "Mission Control PostgREST API"
	}

	stripPropertyDefaults(v3)
	dropInvalidParameters(v3)

	if err := v3.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate converted spec: %w", err)
	}

	return v3, nil
}

// dropInvalidParameters removes parameters with no name from every operation.
// PostgREST occasionally emits placeholder parameter entries (e.g. for RPC
// paths that take no arguments) which the OAS 3.1 validator rejects.
func dropInvalidParameters(spec *openapi3.T) {
	if spec == nil || spec.Paths == nil {
		return
	}
	for _, item := range spec.Paths.Map() {
		if item == nil {
			continue
		}
		item.Parameters = filterNamedParameters(item.Parameters)
		for _, op := range allOperations(item) {
			if op == nil {
				continue
			}
			op.Parameters = filterNamedParameters(op.Parameters)
		}
	}
}

func filterNamedParameters(in openapi3.Parameters) openapi3.Parameters {
	out := in[:0]
	for _, p := range in {
		if p == nil || p.Value == nil || p.Value.Name == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func allOperations(item *openapi3.PathItem) []*openapi3.Operation {
	return []*openapi3.Operation{
		item.Get, item.Post, item.Put, item.Patch, item.Delete,
		item.Head, item.Options, item.Trace,
	}
}

// stripPropertyDefaults removes the `default` value from every component
// schema property. PostgREST exposes Postgres-side SQL expressions (and
// occasionally type-incompatible literals) here that fail OAS 3.1
// validation and provide no value to client codegen.
func stripPropertyDefaults(spec *openapi3.T) {
	if spec == nil || spec.Components == nil {
		return
	}
	for _, schemaRef := range spec.Components.Schemas {
		if schemaRef == nil || schemaRef.Value == nil {
			continue
		}
		for _, propRef := range schemaRef.Value.Properties {
			if propRef == nil || propRef.Value == nil {
				continue
			}
			propRef.Value.Default = nil
		}
	}
}

// Marshal serialises a spec to indented JSON.
func Marshal(spec *openapi3.T) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}
