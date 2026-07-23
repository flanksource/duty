package postgrest

import (
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/flanksource/duty/rbac"
	"github.com/flanksource/duty/rbac/policy"
)

// ExcludedRBACObjects lists the RBAC categories that must never be
// surfaced in the published SDK. Auth/Kratos internals, incident/comment
// data, database system tables, and the legacy topology surface are all
// server-side concerns; exposing them would imply they are part of the
// public contract.
var ExcludedRBACObjects = map[string]bool{
	policy.ObjectAuthConfidential: true,
	policy.ObjectIncident:         true,
	policy.ObjectDatabaseSystem:   true,
	policy.ObjectTopology:         true,
}

// FilterByRBAC removes every path whose owning resource maps to one of the
// excluded RBAC objects, plus the corresponding component schema. Paths
// without an RBAC mapping are left alone (PostgREST exposes some helper
// endpoints, e.g. `/`, that don't belong to a single domain).
func FilterByRBAC(spec *openapi3.T) (removedPaths, removedSchemas int) {
	if spec == nil || spec.Paths == nil {
		return 0, 0
	}
	for path := range spec.Paths.Map() {
		resource := strings.TrimPrefix(path, "/")
		obj := rbac.GetObjectByTable(resource)
		if obj == "" || !ExcludedRBACObjects[obj] {
			continue
		}
		spec.Paths.Delete(path)
		removedPaths++

		if spec.Components != nil && spec.Components.Schemas != nil {
			if _, ok := spec.Components.Schemas[resource]; ok {
				delete(spec.Components.Schemas, resource)
				removedSchemas++
			}
		}
	}
	return removedPaths, removedSchemas
}

// FilterRPCPaths drops `/rpc/*` endpoints that aren't part of the Mission
// Control public surface. The authoritative list of legitimate RPCs is the
// `rpc/...` entries in rbac/objects.go; PostgREST also exposes pgcrypto
// helpers (pgp_key_id, armor, dearmor, ...) and other postgres builtins
// that have no RBAC mapping and shouldn't ship in the SDK.
//
// Rule: keep an RPC iff `rbac.GetObjectByTable("rpc/<name>")` returns a
// non-empty object. Unregistered RPCs are removed.
func FilterRPCPaths(spec *openapi3.T) (removedPaths int) {
	if spec == nil || spec.Paths == nil {
		return 0
	}
	for path := range spec.Paths.Map() {
		if !strings.HasPrefix(path, "/rpc/") {
			continue
		}
		resource := strings.TrimPrefix(path, "/")
		if rbac.GetObjectByTable(resource) != "" {
			continue
		}
		spec.Paths.Delete(path)
		removedPaths++
	}
	return removedPaths
}

// TagWithRBAC attaches `x-rbac-object` and `x-rbac-action` extensions to every
// operation in the spec, and registers each RBAC object as an OpenAPI tag so
// generated clients group endpoints by domain (catalog, canary, playbooks…).
func TagWithRBAC(spec *openapi3.T) {
	if spec == nil || spec.Paths == nil {
		return
	}

	tags := map[string]bool{}

	for path, item := range spec.Paths.Map() {
		resource := strings.TrimPrefix(path, "/")
		obj := rbac.GetObjectByTable(resource)
		if obj == "" {
			continue
		}
		tags[obj] = true

		for method, op := range methodMap(item) {
			if op == nil {
				continue
			}
			action := rbac.GetActionFromHttpMethod(method)
			setExt(op, "x-rbac-object", obj)
			if action != "" {
				setExt(op, "x-rbac-action", action)
			}
			if !slices.Contains(op.Tags, obj) {
				op.Tags = append(op.Tags, obj)
			}
		}
	}

	mergeTags(spec, tags)
}

func setExt(op *openapi3.Operation, key string, value string) {
	if op.Extensions == nil {
		op.Extensions = map[string]any{}
	}
	op.Extensions[key] = value
}

func methodMap(item *openapi3.PathItem) map[string]*openapi3.Operation {
	return map[string]*openapi3.Operation{
		http.MethodGet:    item.Get,
		http.MethodPost:   item.Post,
		http.MethodPatch:  item.Patch,
		http.MethodDelete: item.Delete,
		http.MethodPut:    item.Put,
	}
}

func mergeTags(spec *openapi3.T, names map[string]bool) {
	existing := map[string]bool{}
	for _, t := range spec.Tags {
		existing[t.Name] = true
	}
	added := make([]string, 0, len(names))
	for name := range names {
		if existing[name] {
			continue
		}
		added = append(added, name)
	}
	sort.Strings(added)
	for _, name := range added {
		spec.Tags = append(spec.Tags, &openapi3.Tag{Name: name})
	}
}
