package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Grant admits the rows of a resource type that are in all of its Scopes: Scope and every one of its Constraints.
// Each is a Scope id.
//
// For example, a Role rule reading configs in Scope payments, bound with a constraint on Scope eu, for a request made
// with an access token whose Role reads Scope prod:
//
//	Grant{Scope: "<payments>", Constraints: []string{"<eu>", "<prod>"}}  // configs in payments, eu and prod
type Grant struct {
	// Scope grants the rows: the resource Scope of a Role rule, or a Scope a Permission names.
	Scope string `json:"scope,omitempty"`

	// Constraints narrow Scope: a row must be in every one of them, e.g. the resource Scope of a RoleBinding's
	// constraint, or those of an access token's Roles. Empty when the grant isn't narrowed.
	Constraints []string `json:"constraints,omitempty"`
}

// legacyGrant is a Grant as claims issued before Constraints wrote it.
//
// TODO: remove once those claims have expired.
type legacyGrant struct {
	Grant
	Constraint   string   `json:"constraint,omitempty"`
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

// normalize puts the grant's Scope ids in canonical form, sorts its Constraints and drops a Scope repeated within it.
// It reports false when an id isn't a UUID, or when the grant has no Scope.
func (g Grant) normalize() (Grant, bool) {
	var ok bool
	if g.Scope, ok = canonicalScopeID(g.Scope); !ok {
		return Grant{}, false
	}

	var constraints []string
	for _, id := range g.Constraints {
		canonical, ok := canonicalScopeID(id)
		if !ok {
			return Grant{}, false
		}
		if canonical != g.Scope {
			constraints = append(constraints, canonical)
		}
	}
	slices.Sort(constraints)
	g.Constraints = slices.Compact(constraints)
	return g, true
}

func (g Grant) key() string {
	return g.Scope + "|" + strings.Join(g.Constraints, ",")
}

// Grants are a subject's grants on one resource type: every row (All), or the rows at least one grant admits (Any).
// A type with no grants, or with neither, lists no rows.
//
// They're marshalled into the claim as "all", or as the list of grants:
//
//	"all"
//	[{"scope": "<prod>"}, {"scope": "<payments>", "constraints": ["<eu>"]}]  // rows in prod, or in both payments and eu
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
		ids = append(ids, grant.Scope)
		ids = append(ids, grant.Constraints...)
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

	var grants []legacyGrant
	if err := json.Unmarshal(data, &grants); err != nil {
		return err
	}

	*g = Grants{}
	for _, legacy := range grants {
		grant := legacy.Grant
		if legacy.Constraint != "" {
			grant.Constraints = append(grant.Constraints, legacy.Constraint)
		}
		grant.Constraints = append(grant.Constraints, legacy.Impersonated...)
		g.Add(grant)
	}
	return nil
}
