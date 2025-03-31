package rbac

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/casbin/govaluate"
	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

type Selectors struct {
	Playbooks   []types.ResourceSelector `json:"playbooks,omitempty"`
	Connections []types.ResourceSelector `json:"connections,omitempty"`
	Configs     []types.ResourceSelector `json:"configs,omitempty"`
	Components  []types.ResourceSelector `json:"components,omitempty"`
}

func matchPerm(attr *models.ABACAttribute, _agents any, tagsEncoded string) (bool, error) {
	var rAgents []string
	switch v := _agents.(type) {
	case []any:
		rAgents = lo.Map(v, func(item any, _ int) string { return item.(string) })
	case string:
		if v != "" {
			rAgents = append(rAgents, v)
		}
	}

	rTags := collections.SelectorToMap(tagsEncoded)
	if attr.Config.ID != uuid.Nil {
		var agentsMatch = true
		if len(rAgents) > 0 {
			agentsMatch = lo.Contains(rAgents, attr.Config.AgentID.String())
		}

		tagsmatch := mapContains(rTags, attr.Config.Tags)
		return tagsmatch && agentsMatch, nil
	}

	return false, nil
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
	enforcer.AddFunction("matchPerm", func(args ...any) (any, error) {
		if len(args) != 3 {
			return false, fmt.Errorf("matchPerm needs 3 arguments. got %d", len(args))
		}

		obj := args[0]
		if _, ok := obj.(string); ok {
			// an object is required to satisfy the agents & tags requirement.
			// If a role is passed, we don't match this permission.
			return false, nil
		}

		attr, ok := obj.(*models.ABACAttribute)
		if !ok {
			return false, errors.New("[matchPerm] unknown input type: expected *models.ABACAttribute")
		}

		agents := args[1]
		tags := args[2]

		tagsEncoded, ok := tags.(string)
		if !ok {
			return false, errors.New("tags must be a string")
		}

		return matchPerm(attr, agents, tagsEncoded)
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
}

// mapContains returns true if `request` fully contains `want`.
func mapContains(want map[string]string, request map[string]string) bool {
	for k, v := range want {
		if request[k] != v {
			return false
		}
	}

	return true
}
