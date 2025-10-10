package rbac

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/casbin/govaluate"
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

type ScopeRef struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type Selectors struct {
	Playbooks   []types.ResourceSelector `json:"playbooks,omitempty"`
	Connections []types.ResourceSelector `json:"connections,omitempty"`
	Configs     []types.ResourceSelector `json:"configs,omitempty"`
	Components  []types.ResourceSelector `json:"components,omitempty"`
	Scopes      []ScopeRef               `json:"scopes,omitempty"`
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
	resourcePairs := []resourcePair{
		{attr.Playbook.ID, &attr.Playbook, selector.Playbooks},
		{attr.Component.ID, attr.Component, selector.Components},
		{attr.Connection.ID, &attr.Connection, selector.Connections},
		{attr.Config.ID, attr.Config, selector.Configs},
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

		rs, err := base64.StdEncoding.DecodeString(selector)
		if err != nil {
			return false, err
		}

		var objectSelector Selectors
		if err := json.Unmarshal([]byte(rs), &objectSelector); err != nil {
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
