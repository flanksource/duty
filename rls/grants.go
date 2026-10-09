package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Grant admits the rows of a resource type that are in all of its Scopes: Scope, Constraint when it's set, and every
// Impersonated Scope. Each is a Scope id.
//
// For example, a Role rule reading configs in Scope payments, bound with a constraint on Scope eu, for a request
// limited to Scope prod:
//
//	Grant{Scope: "<payments>", Constraint: "<eu>", Impersonated: []string{"<prod>"}}  // configs in payments, eu and prod
type Grant struct {
	// Scope grants the rows: the resource Scope of a Role rule, or a Scope a Permission names.
	// Empty only for a subject granted every row whose request is limited to Scopes (see Grants.Limit).
	Scope string `json:"scope,omitempty"`

	// Constraint is the resource Scope of the RoleBinding's constraint, which narrows Scope. Empty without one.
	Constraint string `json:"constraint,omitempty"`

	// Impersonated are Scopes the request is limited to (Grants.Limit), which narrow the grant further: a row must
	// be in every one of them. Empty when the request isn't limited.
	Impersonated []string `json:"impersonated,omitempty"`
}

// canonicalScopeID returns the id in canonical form, and false when it isn't a UUID.
func canonicalScopeID(id string) (string, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}

// normalize puts the grant's Scope ids in canonical form and drops a Scope repeated within it. It reports false when
// an id isn't a UUID, when the grant names no Scope, or when it has a Constraint without a Scope.
func (g Grant) normalize() (Grant, bool) {
	var ok bool
	if g.Scope != "" {
		if g.Scope, ok = canonicalScopeID(g.Scope); !ok {
			return Grant{}, false
		}
	}
	if g.Constraint != "" {
		if g.Constraint, ok = canonicalScopeID(g.Constraint); !ok || g.Scope == "" {
			return Grant{}, false
		}
	}

	var impersonated []string
	for _, id := range g.Impersonated {
		canonical, ok := canonicalScopeID(id)
		if !ok {
			return Grant{}, false
		}
		if canonical != g.Scope && canonical != g.Constraint {
			impersonated = append(impersonated, canonical)
		}
	}
	slices.Sort(impersonated)
	g.Impersonated = slices.Compact(impersonated)

	if g.Constraint == g.Scope {
		g.Constraint = ""
	}
	if g.Scope == "" && len(g.Impersonated) == 0 {
		return Grant{}, false
	}
	return g, true
}

func (g Grant) key() string {
	return g.Scope + "|" + g.Constraint + "|" + strings.Join(g.Impersonated, ",")
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

// Add adds a grant. A grant naming anything but Scope ids, or no Scope, is ignored: it admits nothing.
func (g *Grants) Add(grant Grant) {
	grant, ok := grant.normalize()
	if !ok {
		return
	}

	key := grant.key()
	if slices.ContainsFunc(g.Any, func(existing Grant) bool { return existing.key() == key }) {
		return
	}
	g.Any = append(g.Any, grant)
	slices.SortFunc(g.Any, func(a, b Grant) int { return strings.Compare(a.key(), b.key()) })
}

// Limit keeps only the rows in at least one of the Scopes, e.g. those named by the X-Flanksource-Scope header.
// Each grant is split into one grant per Scope, which a row must also be in, and a subject granted every row gets
// one grant per Scope. Limiting again narrows further: a row must then be in one Scope of each limit.
// A limit naming no Scope, or anything but Scope ids, admits nothing. Nil grants stay nil: they admit nothing.
//
//	[{scope: A}].Limit(X, Y) => [{scope: A, impersonated: [X]}, {scope: A, impersonated: [Y]}]
//	"all".Limit(X, Y)        => [{impersonated: [X]}, {impersonated: [Y]}]
func (g *Grants) Limit(scopeIDs ...string) {
	if g == nil {
		return
	}

	current := *g
	*g = Grants{}
	if len(scopeIDs) == 0 {
		return
	}
	for _, id := range scopeIDs {
		if _, ok := canonicalScopeID(id); !ok {
			return
		}
	}

	if current.All {
		current.Any = []Grant{{}}
	}
	for _, grant := range current.Any {
		for _, id := range scopeIDs {
			limited := grant
			limited.Impersonated = append(slices.Clone(grant.Impersonated), id)
			g.Add(limited)
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
		for _, id := range append([]string{grant.Scope, grant.Constraint}, grant.Impersonated...) {
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

// MarshalJSON writes the grants as the claim reads them: "all", or the list of grants.
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
