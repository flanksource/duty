package postgrest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/flanksource/duty/schema/openapi"
)

// EnrichResult summarises what enrichment did so callers can log and act on it.
type EnrichResult struct {
	EnrichedSchemas int
	EnrichedFields  int
	MissingTables   []string // registered tables absent from the spec
	UnknownTables   []string // spec table-shaped entries with no registry hit
}

// Enrich walks every component schema in the spec and copies Go-doc
// descriptions onto matching properties using the registry. It returns a
// summary of what was touched plus any registry-completeness issues.
//
// The merge is best-effort: missing struct fields, json:"-" fields, and
// unregistered components are skipped silently. The completeness check is
// the caller's job (see RegistryCheck).
func Enrich(spec *openapi3.T) (EnrichResult, error) {
	if spec == nil || spec.Components == nil {
		return EnrichResult{}, nil
	}

	res := EnrichResult{}
	seen := map[string]bool{}

	for name, schemaRef := range spec.Components.Schemas {
		if schemaRef == nil || schemaRef.Value == nil {
			continue
		}
		seen[name] = true

		resource, ok := Lookup(name)
		if !ok {
			res.UnknownTables = append(res.UnknownTables, name)
			continue
		}

		descByJSONTag, err := descriptionsFor(resource.GoType)
		if err != nil {
			return res, fmt.Errorf("reflect %s: %w", name, err)
		}

		applied := applyDescriptions(schemaRef.Value, descByJSONTag)
		if applied > 0 {
			res.EnrichedSchemas++
			res.EnrichedFields += applied
		}
	}

	for name := range All() {
		if !seen[name] {
			res.MissingTables = append(res.MissingTables, name)
		}
	}

	return res, nil
}

func applyDescriptions(schema *openapi3.Schema, desc map[string]string) int {
	if schema == nil {
		return 0
	}
	count := 0
	for propName, propRef := range schema.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}
		if propRef.Value.Description != "" {
			continue
		}
		d, ok := desc[propName]
		if !ok || d == "" {
			continue
		}
		propRef.Value.Description = d
		count++
	}
	return count
}

// descriptionsFor reflects a Go struct via the existing duty schema reflector,
// then flattens the resulting JSON Schema so every property is keyed by its
// json tag.
func descriptionsFor(t reflect.Type) (map[string]string, error) {
	prototype := reflect.New(t).Interface()
	raw, err := openapi.GenerateSchema(prototype)
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode reflected schema: %w", err)
	}

	out := map[string]string{}
	root := resolveRoot(doc)
	collectDescriptions(root, doc, out, map[string]bool{})
	return out, nil
}

// resolveRoot returns the top-level schema node, following $ref into
// $defs/definitions if necessary.
func resolveRoot(doc map[string]any) map[string]any {
	if ref, ok := doc["$ref"].(string); ok {
		if target := followRef(ref, doc); target != nil {
			return target
		}
	}
	return doc
}

func followRef(ref string, doc map[string]any) map[string]any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	var node any = doc
	for _, p := range parts {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[p]
	}
	out, _ := node.(map[string]any)
	return out
}

func collectDescriptions(schema map[string]any, doc map[string]any, out map[string]string, visited map[string]bool) {
	if schema == nil {
		return
	}
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range props {
		propSchema, _ := raw.(map[string]any)
		if propSchema == nil {
			continue
		}
		if ref, ok := propSchema["$ref"].(string); ok && !visited[ref] {
			visited[ref] = true
			if target := followRef(ref, doc); target != nil {
				collectDescriptions(target, doc, out, visited)
			}
		}
		if desc, ok := propSchema["description"].(string); ok && desc != "" {
			if _, dup := out[name]; !dup {
				out[name] = desc
			}
		}
	}
}

// RegistryCheck returns an error listing tables present in the live spec but
// missing from the Go registry. Use this in CI to make new tables fail loudly
// until a Register(...) line is added.
func RegistryCheck(unknown []string) error {
	if len(unknown) == 0 {
		return nil
	}
	// Suppress noise from PostgREST views and built-in helper schemas that
	// callers wouldn't model in Go (e.g. openapi-fragment definitions).
	relevant := filterModellable(unknown)
	if len(relevant) == 0 {
		return nil
	}
	return fmt.Errorf("postgrest: %d schema(s) in the live spec are not registered:\n  - %s",
		len(relevant), strings.Join(relevant, "\n  - "))
}

func filterModellable(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if strings.HasPrefix(n, "(") || strings.Contains(n, ".") {
			continue
		}
		out = append(out, n)
	}
	return out
}

