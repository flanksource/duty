package rbac

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/casbin/govaluate"
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

type NamespacedNameIDSelector struct {
	ID        string `json:"id,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type ViewRef NamespacedNameIDSelector

// Selectors represents the object_selector from a permission and specifies
// resource selectors for multiple resource types used in ABAC authorization.
//
// For authorization to succeed, all specified resource type selectors must match
// the corresponding resources in the ABACAttribute. If a selector is specified
// for a resource type but the attribute lacks that resource, authorization fails.
// If an attribute provides a resource but no selector exists for that type, the
// permission is considered non-restrictive for that resource (authorized).
type Selectors struct {
	Playbooks   []types.ResourceSelector `json:"playbooks,omitempty"`
	Connections []types.ResourceSelector `json:"connections,omitempty"`
	Configs     []types.ResourceSelector `json:"configs,omitempty"`
	Components  []types.ResourceSelector `json:"components,omitempty"`
	Views       []ViewRef                `json:"views,omitempty"`
}

type addableEnforcer interface {
	AddFunction(name string, function govaluate.ExpressionFunction)
}

type resourcePair struct {
	attrField    uuid.UUID
	attrResource types.ResourceSelectable
	selectors    []types.ResourceSelector
}

// matchResourceSelector matches an ABACAttribute against resource selectors
func matchResourceSelector(attr *models.ABACAttribute, selector Selectors) (bool, error) {
	// The view selector isn't fully resourceSelector compliant yet.
	// For now we start with just the namespace/name and id selector
	var viewSelectors []types.ResourceSelector
	for _, viewRef := range selector.Views {
		viewSelectors = append(viewSelectors, types.ResourceSelector{
			ID:        viewRef.ID,
			Namespace: viewRef.Namespace,
			Name:      viewRef.Name,
		})
	}

	resourcePairs := []resourcePair{
		{attr.Playbook.ID, &attr.Playbook, selector.Playbooks},
		{attr.Component.ID, attr.Component, selector.Components},
		{attr.Connection.ID, &attr.Connection, selector.Connections},
		{attr.Config.ID, attr.Config, selector.Configs},
		{attr.View.ID, &attr.View, viewSelectors},
	}

	for _, pair := range resourcePairs {
		if !matchResourceSelectorPair(pair) {
			return false, nil
		}
	}

	return true, nil
}

func matchResourceSelectorPair(pair resourcePair) bool {
	if pair.attrField != uuid.Nil {
		if len(pair.selectors) == 0 {
			// An attribute was provided but there's no selector to match it against
			//
			// Essentially, what's happening here is that the permission was not restrictive enough.
			// The selector in the permission doesn't care about this attribute.
			// So it's authorized.
			return true
		}

		// Must match one of the selectors
		for _, rs := range pair.selectors {
			if rs.Matches(pair.attrResource) {
				return true
			}
		}

		// matched none
		return false
	} else if len(pair.selectors) > 0 {
		// A selector was provided but there's no attribute to match it against
		return false
	}

	return true
}

func AddCustomFunctions(enforcer addableEnforcer) {
	// This is used in the Casbin matcher to differentiate between RBAC checks (string objects)
	// and ABAC checks (ABACAttribute objects).
	enforcer.AddFunction("isString", func(args ...any) (any, error) {
		if len(args) != 1 {
			return false, fmt.Errorf("isString needs 1 argument. got %d", len(args))
		}

		_, ok := args[0].(string)
		return ok, nil
	})

	enforcer.AddFunction("matchResourceSelector", func(args ...any) (any, error) {
		if len(args) != 2 {
			return false, fmt.Errorf("matchResourceSelector needs 2 arguments. got %d", len(args))
		}

		attributeSet := args[0]
		if _, ok := attributeSet.(string); ok {
			return false, nil
		}

		attr, ok := attributeSet.(*models.ABACAttribute)
		if !ok {
			return false, fmt.Errorf("[matchResourceSelector] unknown input type: %T. expected *models.ABACAttribute", attributeSet)
		}

		selector, ok := args[1].(string)
		if !ok {
			return false, fmt.Errorf("[matchResourceSelector] selector must be a string")
		}

		if attr == nil {
			return false, errors.New("attribute cannot be nil")
		}

		var objectSelector Selectors
		if err := json.Unmarshal([]byte(selector), &objectSelector); err != nil {
			return false, err
		}

		return matchResourceSelector(attr, objectSelector)
	})

	// str converts UUIDs to strings for comparison in Casbin conditions.
	// We need this because - for ABAC, our attributes may have IDs as uuid.UUID types.
	// for casbin rules where we need to match Id against a <uuid-string>,
	// attribute.ID == "<uuid-string>" will always be false because they are different types.
	//
	// This is useful when creating casbin policies from permissions that are tied to a specific resource by id.
	enforcer.AddFunction("str", func(args ...any) (any, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("str needs 1 argument. got %d", len(args))
		}

		switch v := args[0].(type) {
		case uuid.UUID:
			return v.String(), nil
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	})
}
