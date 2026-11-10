// Package postgrest enriches the OpenAPI spec emitted by PostgREST with
// metadata sourced from the Go domain models and the RBAC object map.
//
// The package is split into small files by responsibility: registry mapping,
// schema conversion, enrichment, RBAC tagging, and RPC overlay application.
package postgrest

import (
	"fmt"
	"maps"
	"reflect"
)

// Kind classifies a PostgREST-exposed resource.
type Kind int

const (
	KindTable Kind = iota
	KindView
	KindRPC
)

func (k Kind) String() string {
	switch k {
	case KindTable:
		return "table"
	case KindView:
		return "view"
	case KindRPC:
		return "rpc"
	}
	return "unknown"
}

// Resource describes a single PostgREST-exposed database object.
type Resource struct {
	Name   string
	Kind   Kind
	GoType reflect.Type
}

var registry = map[string]Resource{}

// Register binds a PostgREST resource name to a Go prototype struct. Pass a
// pointer to a zero-valued struct so reflection can resolve the concrete type.
//
// Example:
//
//	postgrest.Register("config_items", postgrest.KindTable, &models.ConfigItem{})
func Register(name string, kind Kind, prototype any) {
	if name == "" {
		panic("postgrest.Register: name must not be empty")
	}
	if prototype == nil {
		panic(fmt.Sprintf("postgrest.Register(%q): prototype must not be nil", name))
	}

	t := reflect.TypeOf(prototype)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	registry[name] = Resource{Name: name, Kind: kind, GoType: t}
}

// Lookup returns the Resource bound to name, if any.
func Lookup(name string) (Resource, bool) {
	r, ok := registry[name]
	return r, ok
}

// All returns a snapshot of every registered Resource keyed by name.
func All() map[string]Resource {
	out := make(map[string]Resource, len(registry))
	maps.Copy(out, registry)
	return out
}
