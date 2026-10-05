package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Grants are a subject's grants on one resource type: all of its rows, or the rows that are in every Scope
// of at least one grant. A grant is a set of Scope ids; it's never empty and never repeats a Scope.
//
// It's marshalled as "all", or as an array of grants, e.g. [["<scope id>", "<scope id>"], ["<scope id>"]].
// Claims name Scopes only: each listing reads the Scopes' membership when it runs.
type Grants struct {
	All  bool
	Sets [][]string
}

// AllRows grants every row of a type.
func AllRows() *Grants {
	return &Grants{All: true}
}

// NoRows grants no row of a type.
func NoRows() *Grants {
	return &Grants{}
}

// Add adds a grant: the rows that are in every one of the Scopes. An empty grant is ignored.
func (g *Grants) Add(scopeIDs ...string) {
	set := normalize(scopeIDs)
	if len(set) == 0 {
		return
	}

	for _, existing := range g.Sets {
		if slices.Equal(existing, set) {
			return
		}
	}
	g.Sets = append(g.Sets, set)
	slices.SortFunc(g.Sets, func(a, b []string) int { return strings.Compare(strings.Join(a, ","), strings.Join(b, ",")) })
}

// Narrow requires every row to also be in each of the Scopes: they're added to every grant.
// Rows of all of a type become the rows in the Scopes.
func (g *Grants) Narrow(scopeIDs ...string) {
	extra := normalize(scopeIDs)
	if len(extra) == 0 {
		return
	}

	if g.All {
		g.All = false
		g.Sets = [][]string{extra}
		return
	}

	sets := g.Sets
	g.Sets = nil
	for _, set := range sets {
		g.Add(append(slices.Clone(set), extra...)...)
	}
}

// IsEmpty reports whether no row is granted.
func (g *Grants) IsEmpty() bool {
	return g == nil || (!g.All && len(g.Sets) == 0)
}

// ScopeIDs returns every Scope the grants name, sorted.
func (g *Grants) ScopeIDs() []string {
	if g == nil {
		return nil
	}
	var ids []string
	for _, set := range g.Sets {
		ids = append(ids, set...)
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

	parts := make([]string, 0, len(g.Sets))
	for _, set := range g.Sets {
		parts = append(parts, strings.Join(set, "+"))
	}
	slices.Sort(parts)
	return "[" + strings.Join(parts, ",") + "]"
}

func (g Grants) MarshalJSON() ([]byte, error) {
	if g.All {
		return json.Marshal("all")
	}
	if g.Sets == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(g.Sets)
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

	var sets [][]string
	if err := json.Unmarshal(data, &sets); err != nil {
		return err
	}

	*g = Grants{}
	for _, set := range sets {
		g.Add(set...)
	}
	return nil
}

func normalize(ids []string) []string {
	var set []string
	for _, id := range ids {
		if id = strings.TrimSpace(strings.ToLower(id)); id != "" {
			set = append(set, id)
		}
	}
	slices.Sort(set)
	return slices.Compact(set)
}
