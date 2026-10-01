package rbac

import (
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/types"
)

// RuleResources selects resources by type, like a Scope.
// A resource belongs to it when a selector of the resource's own type matches the resource.
type RuleResources map[string][]types.ResourceSelector

// RuleCondition is the condition of a casbin policy compiled from a Role rule.
//
// It carries the contract of the rule's action, so a request that doesn't fit the contract
// never matches an allow rule, and always matches a deny rule.
type RuleCondition struct {
	// ResourceTypes are the resource types the action accepts as its primary resource.
	ResourceTypes []string `json:"resourceTypes"`

	// Resource selects the primary resources: a resource must belong to every selection,
	// e.g. the rule's Scope and the Scope a RoleBinding narrows it with.
	Resource []RuleResources `json:"resource"`

	// TargetTypes are the resource types the action accepts as a target.
	// Empty when the action doesn't take a target.
	TargetTypes []string `json:"targetTypes,omitempty"`

	// Target selects the targets: a target must belong to every selection.
	// When empty, the rule only matches requests without a target.
	Target []RuleResources `json:"target,omitempty"`

	// Deny is set for deny rules. A request that doesn't fit the contract,
	// or whose selectors can't be evaluated, matches a deny rule.
	Deny bool `json:"deny,omitempty"`
}

type requestResource struct {
	kind     string
	resource types.ResourceSelectable
}

// requestResources returns the resources an authorization request involves: the ones with an id.
func requestResources(attr *models.ABACAttribute) []requestResource {
	var resources []requestResource
	add := func(kind string, id uuid.UUID, resource types.ResourceSelectable) {
		if id != uuid.Nil {
			resources = append(resources, requestResource{kind: kind, resource: resource})
		}
	}

	add(policy.ResourceConfig, attr.Config.ID, attr.Config)
	add(policy.ResourceComponent, attr.Component.ID, attr.Component)
	add(policy.ResourceCheck, attr.Check.ID, attr.Check)
	add(policy.ResourceCanary, attr.Canary.ID, attr.Canary)
	add(policy.ResourcePlaybook, attr.Playbook.ID, &attr.Playbook)
	add(policy.ResourceView, attr.View.ID, attr.View)
	add(policy.ResourceConnection, attr.Connection.ID, &attr.Connection)
	return resources
}

// matchRule reports whether a Role rule matches a request.
//
// The request must involve exactly one primary resource and at most one target, of the types the
// action accepts, and nothing else. The primary resource must belong to every selection of the rule's
// resource, and the target to every selection of the rule's target. A rule without a target only matches
// requests without one.
func matchRule(attr *models.ABACAttribute, condition RuleCondition) bool {
	var primary, targets []requestResource
	for _, resource := range requestResources(attr) {
		switch {
		case slices.Contains(condition.ResourceTypes, resource.kind):
			primary = append(primary, resource)
		case slices.Contains(condition.TargetTypes, resource.kind):
			targets = append(targets, resource)
		default:
			return condition.Deny
		}
	}

	if len(primary) != 1 || len(targets) > 1 || len(condition.Resource) == 0 {
		return condition.Deny
	}

	if matched, err := belongsToAll(primary[0], condition.Resource); err != nil {
		return condition.Deny
	} else if !matched {
		return false
	}

	if len(condition.Target) == 0 {
		return len(targets) == 0
	} else if len(targets) == 0 {
		return false
	}

	matched, err := belongsToAll(targets[0], condition.Target)
	if err != nil {
		return condition.Deny
	}
	return matched
}

// belongsToAll reports whether the resource belongs to every selection.
// An error is returned only when no selection rules the resource out and some couldn't be evaluated.
func belongsToAll(resource requestResource, selections []RuleResources) (bool, error) {
	var errs []error
	for _, selection := range selections {
		if matched, err := selection.matches(resource); err != nil {
			errs = append(errs, err)
		} else if !matched {
			return false, nil
		}
	}

	if len(errs) > 0 {
		return false, errors.Join(errs...)
	}
	return true, nil
}

// matches reports whether the resource belongs to the selection.
// An error is returned only when no selector matches and some couldn't be evaluated.
func (r RuleResources) matches(resource requestResource) (bool, error) {
	var errs []error
	for _, selector := range r[resource.kind] {
		if matched, err := selector.Matches(resource.resource); err != nil {
			errs = append(errs, err)
		} else if matched {
			return true, nil
		}
	}

	return false, errors.Join(errs...)
}
