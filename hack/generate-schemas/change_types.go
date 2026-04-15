package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/flanksource/duty/schema/openapi"
	"github.com/flanksource/duty/types"
)

// typesImportPath is the Go import path whose source files carry the doc
// comments that describe the ConfigChangeDetail variants. The schema
// generator runs inside hack/generate-schemas (package main), so the
// reflector can't locate those comments by walking the reflected type's own
// package — we pass this path explicitly.
const typesImportPath = "github.com/flanksource/duty/types"

// schemaRootName is the $defs key and $ref target that downstream tooling
// (e.g. incident-commander report builder) looks up in the generated schema.
const schemaRootName = "ConfigChangeDetailsSchema"

// schemaID is the canonical $id embedded in the generated schema.
const schemaID = "https://github.com/flanksource/duty/types/config-change-details-schema"

// schemaDraft is the JSON schema dialect the generator emits.
const schemaDraft = "https://json-schema.org/draft/2020-12/schema"

// changeDetailsEnvelope is a synthetic wrapper whose fields hold one pointer
// per ConfigChangeDetail variant. Reflecting this envelope causes
// invopop/jsonschema to emit shared $defs for every variant and all nested
// types (Identity, Event, Environment, ...), which the second pass then
// specializes with the kind-discriminator constant.
type changeDetailsEnvelope struct {
	UserChange       *types.UserChangeDetails       `json:"userChange,omitempty"`
	Screenshot       *types.ScreenshotDetails       `json:"screenshot,omitempty"`
	PermissionChange *types.PermissionChangeDetails `json:"permissionChange,omitempty"`
	GroupMembership  *types.GroupMembership         `json:"groupMembership,omitempty"`
	Identity         *types.Identity                `json:"identity,omitempty"`
	Approval         *types.Approval                `json:"approval,omitempty"`
	GitSource        *types.GitSource               `json:"gitSource,omitempty"`
	HelmSource       *types.HelmSource              `json:"helmSource,omitempty"`
	ImageSource      *types.ImageSource             `json:"imageSource,omitempty"`
	DatabaseSource   *types.DatabaseSource          `json:"databaseSource,omitempty"`
	Source           *types.Source                  `json:"source,omitempty"`
	Environment      *types.Environment             `json:"environment,omitempty"`
	Event            *types.Event                   `json:"event,omitempty"`
	Test             *types.Test                    `json:"test,omitempty"`
	Promotion        *types.Promotion               `json:"promotion,omitempty"`
	PipelineRun      *types.PipelineRun             `json:"pipelineRun,omitempty"`
	Change           *types.Change                  `json:"change,omitempty"`
	ConfigChange     *types.ConfigChange            `json:"configChange,omitempty"`
	Restore          *types.Restore                 `json:"restore,omitempty"`
	Backup           *types.Backup                  `json:"backup,omitempty"`
	Dimension        *types.Dimension               `json:"dimension,omitempty"`
	Scale            *types.Scale                   `json:"scale,omitempty"`
}

// generateChangeTypesSchema produces the kind-discriminated union schema for
// ConfigChangeDetails and writes it to outPath. The result is a deterministic,
// stable JSON document suitable for checking into the repo.
func generateChangeTypesSchema(outPath string) error {
	defs, err := reflectSharedDefs()
	if err != nil {
		return err
	}

	detailTypeNames, err := detailTypeNames()
	if err != nil {
		return err
	}

	if err := injectKindDiscriminators(defs, detailTypeNames); err != nil {
		return err
	}

	rootExamples, err := marshalExamples(types.ConfigChangeExamples)
	if err != nil {
		return err
	}

	rootDef := buildRootDef(detailTypeNames, rootExamples)
	defs[schemaRootName] = rootDef

	if err := attachExamplesToDefs(defs, types.ConfigChangeExamples); err != nil {
		return err
	}

	root := map[string]any{
		"$schema": schemaDraft,
		"$id":     schemaID,
		"$ref":    "#/$defs/" + schemaRootName,
		"$defs":   defs,
	}

	encoded, err := marshalIndentStable(root)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outPath, encoded, 0644); err != nil {
		return fmt.Errorf("write change-types schema: %w", err)
	}
	return nil
}

// reflectSharedDefs runs pass 1: it reflects changeDetailsEnvelope through the
// openapi generator (loading doc comments from the real types package) and
// returns the $defs map ready for post-processing.
func reflectSharedDefs() (map[string]json.RawMessage, error) {
	raw, err := openapi.GenerateSchemaWithCommentsFrom(&changeDetailsEnvelope{}, typesImportPath)
	if err != nil {
		return nil, fmt.Errorf("reflect change details envelope: %w", err)
	}

	var parsed struct {
		Defs map[string]json.RawMessage `json:"$defs"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse reflected schema: %w", err)
	}
	if parsed.Defs == nil {
		return nil, fmt.Errorf("reflected schema has no $defs")
	}

	// The synthetic envelope leaks into $defs as either ChangeDetailsEnvelope
	// or changeDetailsEnvelope depending on reflector internals. Strip both
	// so the published schema stays private-implementation-free.
	delete(parsed.Defs, "ChangeDetailsEnvelope")
	delete(parsed.Defs, "changeDetailsEnvelope")
	return parsed.Defs, nil
}

// detailTypeNames returns a deterministic (alphabetical) slice of
// {typeName, kind} pairs built from types.ConfigChangeDetailTypes.
type detailTypeName struct {
	TypeName string
	Kind     string
}

func detailTypeNames() ([]detailTypeName, error) {
	out := make([]detailTypeName, 0, len(types.ConfigChangeDetailTypes))
	seen := map[string]bool{}
	for _, d := range types.ConfigChangeDetailTypes {
		name := reflect.TypeOf(d).Name()
		if name == "" {
			return nil, fmt.Errorf("ConfigChangeDetailTypes contains anonymous type %T", d)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate detail type %s in ConfigChangeDetailTypes", name)
		}
		seen[name] = true
		out = append(out, detailTypeName{TypeName: name, Kind: d.Kind()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TypeName < out[j].TypeName })
	return out, nil
}

// injectKindDiscriminators mutates every detail $def so it has a required
// "kind" property with a const matching the Go type's Kind() value. This is
// what turns the plain reflected structs into a discriminated union.
func injectKindDiscriminators(defs map[string]json.RawMessage, names []detailTypeName) error {
	for _, n := range names {
		raw, ok := defs[n.TypeName]
		if !ok {
			return fmt.Errorf("missing $def for %s in reflected schema", n.TypeName)
		}

		var def map[string]any
		if err := json.Unmarshal(raw, &def); err != nil {
			return fmt.Errorf("parse $def for %s: %w", n.TypeName, err)
		}

		props, _ := def["properties"].(map[string]any)
		if props == nil {
			props = map[string]any{}
		}
		props["kind"] = map[string]any{
			"type":        "string",
			"const":       n.Kind,
			"description": "Discriminator that identifies the ConfigChangeDetail variant.",
		}
		def["properties"] = props

		required := collectRequired(def["required"])
		if !slices.Contains(required, "kind") {
			required = append([]string{"kind"}, required...)
		}
		def["required"] = required

		encoded, err := json.Marshal(def)
		if err != nil {
			return fmt.Errorf("re-encode $def for %s: %w", n.TypeName, err)
		}
		defs[n.TypeName] = encoded
	}
	return nil
}

// buildRootDef composes the top-level oneOf union referenced from $ref.
func buildRootDef(names []detailTypeName, examples []json.RawMessage) json.RawMessage {
	oneOf := make([]map[string]any, 0, len(names))
	for _, n := range names {
		oneOf = append(oneOf, map[string]any{"$ref": "#/$defs/" + n.TypeName})
	}

	rootDef := map[string]any{
		"type":        "object",
		"description": "Kind-discriminated union of every ConfigChangeDetail variant. The concrete variant is selected by the required kind constant.",
		"oneOf":       oneOf,
		"examples":    examples,
	}
	encoded, err := json.Marshal(rootDef)
	if err != nil {
		// Can't fail for a map of concrete types/strings.
		panic(fmt.Errorf("marshal root def: %w", err))
	}
	return encoded
}

// marshalExamples marshals each ConfigChangeDetail example via its own
// MarshalJSON, which injects the "kind" field naturally.
func marshalExamples(examples []types.ConfigChangeDetail) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(examples))
	for i, ex := range examples {
		raw, err := json.Marshal(ex)
		if err != nil {
			return nil, fmt.Errorf("marshal example %d (%T): %w", i, ex, err)
		}
		out = append(out, raw)
	}
	return out, nil
}

// attachExamplesToDefs groups examples by their Go type name and writes the
// resulting slice onto the matching $def so consumers can discover a
// per-variant example without having to filter the top-level list.
func attachExamplesToDefs(defs map[string]json.RawMessage, examples []types.ConfigChangeDetail) error {
	grouped := map[string][]json.RawMessage{}
	for i, ex := range examples {
		typeName := reflect.TypeOf(ex).Name()
		raw, err := json.Marshal(ex)
		if err != nil {
			return fmt.Errorf("marshal example %d (%T): %w", i, ex, err)
		}
		grouped[typeName] = append(grouped[typeName], raw)
	}

	for typeName, examples := range grouped {
		raw, ok := defs[typeName]
		if !ok {
			continue
		}
		var def map[string]any
		if err := json.Unmarshal(raw, &def); err != nil {
			return fmt.Errorf("parse $def for %s: %w", typeName, err)
		}
		def["examples"] = examples
		encoded, err := json.Marshal(def)
		if err != nil {
			return fmt.Errorf("re-encode $def for %s: %w", typeName, err)
		}
		defs[typeName] = encoded
	}
	return nil
}

// marshalIndentStable serializes the root schema with 2-space indentation and
// a deterministic key order so the committed file is stable across runs.
func marshalIndentStable(v any) ([]byte, error) {
	canonical, err := canonicalizeValue(v)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(canonical, "", "  ")
}

// canonicalizeValue walks v and converts every map into a key-sorted
// *orderedMap so encoding/json emits fields in a stable, human-friendly order.
// We can't rely on encoding/json's map sorting alone because nested
// json.RawMessage values are re-emitted verbatim — we have to parse them.
func canonicalizeValue(v any) (any, error) {
	switch val := v.(type) {
	case json.RawMessage:
		var parsed any
		if err := json.Unmarshal(val, &parsed); err != nil {
			return nil, err
		}
		return canonicalizeValue(parsed)
	case map[string]any:
		out := newOrderedMap()
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child, err := canonicalizeValue(val[k])
			if err != nil {
				return nil, err
			}
			out.set(k, child)
		}
		return out, nil
	case map[string]json.RawMessage:
		out := newOrderedMap()
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child, err := canonicalizeValue(val[k])
			if err != nil {
				return nil, err
			}
			out.set(k, child)
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			child, err := canonicalizeValue(elem)
			if err != nil {
				return nil, err
			}
			out[i] = child
		}
		return out, nil
	case []map[string]any:
		out := make([]any, len(val))
		for i, elem := range val {
			child, err := canonicalizeValue(elem)
			if err != nil {
				return nil, err
			}
			out[i] = child
		}
		return out, nil
	case []json.RawMessage:
		out := make([]any, len(val))
		for i, elem := range val {
			child, err := canonicalizeValue(elem)
			if err != nil {
				return nil, err
			}
			out[i] = child
		}
		return out, nil
	}
	return v, nil
}

// orderedMap preserves insertion order so canonicalized output is
// deterministic across Go runtime versions (map iteration is randomised, but
// encoding/json emits struct keys in declaration order).
type orderedMap struct {
	keys   []string
	values map[string]any
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: map[string]any{}}
}

func (m *orderedMap) set(k string, v any) {
	if _, ok := m.values[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.values[k] = v
}

func (m *orderedMap) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(m.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return []byte(buf.String()), nil
}

func collectRequired(v any) []string {
	switch val := v.(type) {
	case []string:
		return append([]string{}, val...)
	case []any:
		out := make([]string, 0, len(val))
		for _, elem := range val {
			if s, ok := elem.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

