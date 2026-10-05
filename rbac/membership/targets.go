// Package membership stores which Scopes each resource belongs to.
//
// A Scope's targets are stored as rows of values (scope_targets), and one SQL predicate, scope_target_matches,
// decides whether a resource matches a target. A resource is matched by trigger in the transaction that writes it,
// and a Scope is rebuilt in the transaction that saves it. Checks and listings only read the stored result
// (scope_members), e.g. through ForOperation.
package membership

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/types"
)

// Target is one target of a Scope: a selector over the resources of one type, with its agent resolved to an id.
type Target struct {
	Type     string                 `json:"type"`
	Selector types.ResourceSelector `json:"selector"`
}

// WholeType reports whether the target selects every resource of its type: name "*" and nothing else.
func (t Target) WholeType() bool {
	return t.Selector.Wildcard() && t.Selector.Functions.ComponentConfigTraversal == nil
}

// fields are the selector fields each resource type has, besides id, name and namespace.
var fields = map[string]struct{ agent, types, tags, labels bool }{
	policy.ResourceConfig:     {agent: true, types: true, tags: true, labels: true},
	policy.ResourceComponent:  {agent: true, types: true, labels: true},
	policy.ResourceCheck:      {agent: true, types: true, labels: true},
	policy.ResourceCanary:     {agent: true, labels: true},
	policy.ResourcePlaybook:   {},
	policy.ResourceConnection: {types: true},
}

// Types are the resource types whose membership is stored.
func Types() []string {
	kinds := make([]string, 0, len(fields))
	for kind := range fields {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	return kinds
}

// Supported reports whether the membership of the resource type is stored.
func Supported(kind string) bool {
	_, ok := fields[kind]
	return ok
}

// targetRow is a row of scope_targets. A condition the target doesn't set is nil, and matches anything.
type targetRow struct {
	ScopeID      uuid.UUID `gorm:"column:scope_id"`
	ResourceType string    `gorm:"column:resource_type"`
	ResourceID   *string   `gorm:"column:resource_id"`
	Name         *string   `gorm:"column:name"`
	NamePrefix   *string   `gorm:"column:name_prefix"`
	Namespace    *string   `gorm:"column:namespace"`
	AgentID      *string   `gorm:"column:agent_id"`
	Types        *string   `gorm:"column:types"`
	Tags         *string   `gorm:"column:tags"`
	Labels       *string   `gorm:"column:labels"`
}

// key identifies the row's conditions, for comparing a Scope's stored targets with new ones.
func (r targetRow) key() string {
	raw, _ := json.Marshal([]any{r.ResourceType, r.ResourceID, r.Name, r.NamePrefix, r.Namespace, r.AgentID, r.Types, r.Tags, r.Labels})
	return string(raw)
}

// row converts a target that isn't a whole-type target to a row of scope_targets.
func (t Target) row(scopeID uuid.UUID) (targetRow, error) {
	f, ok := fields[t.Type]
	if !ok {
		return targetRow{}, fmt.Errorf("membership of %s isn't stored", t.Type)
	}

	s := t.Selector
	if s.Search != "" || s.FieldSelector != "" || s.Scope != "" || len(s.Statuses) > 0 || s.Health != "" ||
		s.Functions.ComponentConfigTraversal != nil {
		return targetRow{}, fmt.Errorf("%s selector: search, fieldSelector, scope, statuses, health and functions aren't supported", t.Type)
	}
	if s.ID == "" && s.Name == "" && s.Namespace == "" && s.Agent == "" && len(s.Types) == 0 &&
		s.TagSelector == "" && s.LabelSelector == "" {
		return targetRow{}, fmt.Errorf("an empty %s selector selects nothing", t.Type)
	}

	row := targetRow{ScopeID: scopeID, ResourceType: t.Type}
	optional := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}

	if s.ID != "" {
		id, err := uuid.Parse(s.ID)
		if err != nil {
			return targetRow{}, fmt.Errorf("id %q isn't a uuid", s.ID)
		}
		row.ResourceID = optional(id.String())
	}

	switch {
	case s.Name == "" || s.Name == "*":
	case strings.Count(s.Name, "*") == 1 && strings.HasSuffix(s.Name, "*") && len(s.Name) > 1:
		row.NamePrefix = optional(strings.TrimSuffix(s.Name, "*"))
	case strings.Contains(s.Name, "*"):
		return targetRow{}, fmt.Errorf("name %q: * is only supported at the end", s.Name)
	default:
		row.Name = optional(s.Name)
	}

	if strings.Contains(s.Namespace, "*") {
		return targetRow{}, fmt.Errorf("namespace %q must be exact", s.Namespace)
	}
	row.Namespace = optional(s.Namespace)

	if s.Agent != "" {
		if !f.agent {
			return targetRow{}, fmt.Errorf("%s has no agent", t.Type)
		}
		id, err := uuid.Parse(s.Agent)
		if err != nil {
			return targetRow{}, fmt.Errorf("agent %q must be resolved to an id", s.Agent)
		}
		row.AgentID = optional(id.String())
	}

	if len(s.Types) > 0 {
		if !f.types {
			return targetRow{}, fmt.Errorf("%s has no types", t.Type)
		}
		values := slices.Clone(s.Types)
		slices.Sort(values)
		array := textArray(slices.Compact(values))
		row.Types = &array
	}

	for _, sel := range []struct {
		name, value string
		has         bool
		dest        **string
	}{
		{"tags", s.TagSelector, f.tags, &row.Tags},
		{"labels", s.LabelSelector, f.labels, &row.Labels},
	} {
		if sel.value == "" {
			continue
		} else if !sel.has {
			return targetRow{}, fmt.Errorf("%s has no %s", t.Type, sel.name)
		}

		pairs, err := Equalities(sel.value)
		if err != nil {
			return targetRow{}, fmt.Errorf("%s: %w", sel.name, err)
		} else if pairs == nil {
			// Two values for one key: nothing matches. No resource has a tag with an empty key.
			pairs = map[string]string{"": ""}
		}
		raw, _ := json.Marshal(pairs)
		*sel.dest = optional(string(raw))
	}

	return row, nil
}

// Validate checks that the target can be stored.
func Validate(t Target) error {
	if !Supported(t.Type) {
		return fmt.Errorf("membership of %s isn't stored", t.Type)
	} else if t.WholeType() {
		return nil
	}
	_, err := t.row(uuid.Nil)
	return err
}

// Equalities returns the key=value pairs of a tag or label selector. Every other operator is rejected.
// It returns nil pairs when two pairs give one key different values, which no resource matches.
func Equalities(selector string) (map[string]string, error) {
	parsed, err := labels.Parse(selector)
	if err != nil {
		return nil, err
	}

	requirements, _ := parsed.Requirements()
	pairs := map[string]string{}
	for _, r := range requirements {
		if (r.Operator() != selection.Equals && r.Operator() != selection.DoubleEquals) || r.Values().Len() != 1 {
			return nil, fmt.Errorf("%q must only use key=value pairs", selector)
		}
		value := r.Values().List()[0]
		if existing, ok := pairs[r.Key()]; ok && existing != value {
			return nil, nil
		}
		pairs[r.Key()] = value
	}
	return pairs, nil
}
