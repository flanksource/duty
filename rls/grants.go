package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Grant admits the rows of a resource type that are in all of its Scopes: Scope, and Constraint and Impersonated
// when they're set. Each field is a Scope id.
//
// For example, a Role rule reading configs in Scope payments, bound with a constraint on Scope eu:
//
//	Grant{Scope: "<payments>", Constraint: "<eu>"}  // configs in both payments and eu
type Grant struct {
	// Scope grants the rows: the resource Scope of a Role rule, or a Scope a Permission names.
	Scope string `json:"scope"`

	// Constraint is the resource Scope of the RoleBinding's constraint, which narrows Scope. Empty without one.
	Constraint string `json:"constraint,omitempty"`

	// Impersonated is a Scope named by the X-Flanksource-Scope header, which narrows the grant further.
	// Empty when the request doesn't impersonate.
	Impersonated string `json:"impersonated,omitempty"`
}

// normalize lowercases the grant's Scope ids and drops a Scope repeated within it. It reports false when a Scope id
// isn't a UUID, or Scope is empty, so the grant can't be used.
func (g Grant) normalize() (Grant, bool) {
	for _, id := range []*string{&g.Scope, &g.Constraint, &g.Impersonated} {
		*id = strings.ToLower(strings.TrimSpace(*id))
		if *id == "" {
			continue
		} else if _, err := uuid.Parse(*id); err != nil {
			return Grant{}, false
		}
	}

	if g.Scope == "" {
		return Grant{}, false
	}
	if g.Constraint == g.Scope {
		g.Constraint = ""
	}
	if g.Impersonated == g.Scope || g.Impersonated == g.Constraint {
		g.Impersonated = ""
	}
	return g, true
}

func (g Grant) key() string {
	return g.Scope + "|" + g.Constraint + "|" + g.Impersonated
}

// Grants are a subject's grants on one resource type: every row (All), or the rows at least one grant admits (Any).
// A type with no grants, or with neither, lists no rows.
//
// They're marshalled into the claim as "all", or as the list of grants:
//
//	"all"
//	[{"scope": "<prod>"}, {"scope": "<payments>", "constraint": "<eu>"}]  // rows in prod, or in both payments and eu
type Grants struct {
	// All admits every row of the type.
	All bool

	// Any admits a row that at least one grant admits.
	Any []Grant
}

// AllRows grants every row of a type.
func AllRows() *Grants {
	return &Grants{All: true}
}

// NoRows grants no row of a type.
func NoRows() *Grants {
	return &Grants{}
}

// Add adds a grant. A grant naming anything but Scope ids, or without a Scope, is ignored: it admits nothing.
func (g *Grants) Add(grant Grant) {
	grant, ok := grant.normalize()
	if !ok {
		return
	}

	if slices.ContainsFunc(g.Any, func(existing Grant) bool { return existing == grant }) {
		return
	}
	g.Any = append(g.Any, grant)
	slices.SortFunc(g.Any, func(a, b Grant) int { return strings.Compare(a.key(), b.key()) })
}

// Impersonate narrows the grants to the Scopes named by the X-Flanksource-Scope header: a row must also be in one of
// them. Each grant is split into one grant per Scope, and a subject granted every row gets one grant per Scope.
// Impersonating no Scope admits nothing.
//
//	[{scope: A}].Impersonate(X, Y) => [{scope: A, impersonated: X}, {scope: A, impersonated: Y}]
//	"all".Impersonate(X, Y)        => [{scope: X}, {scope: Y}]
func (g *Grants) Impersonate(scopeIDs ...string) {
	current := *g
	*g = Grants{}

	for _, id := range scopeIDs {
		if current.All {
			g.Add(Grant{Scope: id})
			continue
		}
		for _, grant := range current.Any {
			// Impersonating twice can't be written as one grant, so it admits nothing rather than widen
			if grant.Impersonated != "" && !strings.EqualFold(grant.Impersonated, id) {
				continue
			}
			grant.Impersonated = id
			g.Add(grant)
		}
	}
}

// IsEmpty reports whether no row is granted.
func (g *Grants) IsEmpty() bool {
	return g == nil || (!g.All && len(g.Any) == 0)
}

// ScopeIDs returns every Scope the grants name, sorted.
func (g *Grants) ScopeIDs() []string {
	if g == nil {
		return nil
	}
	var ids []string
	for _, grant := range g.Any {
		for _, id := range []string{grant.Scope, grant.Constraint, grant.Impersonated} {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

// Fingerprint identifies the grants.
func (g *Grants) Fingerprint() string {
	if g == nil {
		return "none"
	} else if g.All {
		return "all"
	}

	parts := make([]string, 0, len(g.Any))
	for _, grant := range g.Any {
		parts = append(parts, grant.key())
	}
	slices.Sort(parts)
	return "[" + strings.Join(parts, ",") + "]"
}

func (g Grants) MarshalJSON() ([]byte, error) {
	if g.All {
		return json.Marshal("all")
	}
	if g.Any == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(g.Any)
}

func (g *Grants) UnmarshalJSON(data []byte) error {
	var all string
	if err := json.Unmarshal(data, &all); err == nil {
		if all != "all" {
			return fmt.Errorf(`grants must be "all" or a list of grants, not %q`, all)
		}
		*g = Grants{All: true}
		return nil
	}

	var grants []Grant
	if err := json.Unmarshal(data, &grants); err != nil {
		return err
	}

	*g = Grants{}
	for _, grant := range grants {
		g.Add(grant)
	}
	return nil
}
