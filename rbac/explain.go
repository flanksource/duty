package rbac

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// ExplainedRule is a rule of the policy that a check could match.
type ExplainedRule struct {
	// Principal is who the rule is filed under: the subject itself, or a role the subject holds.
	Principal string `json:"principal"`

	// Paths are the chains of roles through which the subject holds Principal, each with Principal last.
	// It's empty when the rule is filed under the subject itself.
	Paths [][]string `json:"paths,omitempty"`

	Object    string `json:"object"`
	Action    string `json:"action"`
	Effect    string `json:"effect"`
	Condition string `json:"condition,omitempty"`

	// ID is the id the rule was loaded with, e.g. the binding and rule that produced it.
	ID string `json:"id,omitempty"`

	// Matches reports whether the rule matches the check. A rule that doesn't is for the same action and object,
	// but its condition doesn't hold, e.g. the resource isn't in the rule's Scopes.
	Matches bool `json:"matches"`
}

// maxRolePaths bounds the chains of roles ExplainCheck lists for a subject: beyond it, each role keeps the shortest.
const maxRolePaths = 1000

// ExplainCheck returns the rules that could decide a check of the action by the subject on the object: every rule the
// subject holds, directly or through its roles, for the action and object. Each says whether it matches.
// The check is allowed when some matching rule allows it and none denies it.
//
// object is an RBAC object, e.g. "catalog", or a *models.ABACAttribute, whose Scope membership is read for the
// action as HasPermission reads it.
func ExplainCheck(ctx context.Context, subject string, object any, action string) ([]ExplainedRule, error) {
	if enforcer == nil {
		return nil, nil
	}

	switch o := object.(type) {
	case string:
	case *models.ABACAttribute:
		object = withMembership(ctx, o, action)
	default:
		return nil, fmt.Errorf("unsupported object %T", object)
	}

	principals, paths, err := rolePaths(subject)
	if err != nil {
		return nil, err
	}

	// Each candidate is filed under a subject of its own in a separate enforcer, so a check as that subject can only
	// match that rule.
	m, err := model.NewModelFromString(DefaultModel)
	if err != nil {
		return nil, err
	}
	isolated, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}
	AddCustomFunctions(isolated)

	var candidates []ExplainedRule
	for _, principal := range principals {
		rules, err := enforcer.GetFilteredPolicy(0, principal)
		if err != nil {
			return nil, err
		}

		for _, rule := range rules {
			candidate, ok := candidateRule(principal, paths[principal], rule, object, action)
			if !ok {
				continue
			}

			key := strconv.Itoa(len(candidates))
			if _, err := isolated.AddPolicy(key, candidate.Object, candidate.Action, candidate.Effect, candidate.Condition, candidate.ID); err != nil {
				return nil, err
			}
			candidates = append(candidates, candidate)
		}
	}

	for i := range candidates {
		_, explain, err := isolated.EnforceEx(strconv.Itoa(i), object, action)
		if err != nil {
			return nil, fmt.Errorf("failed to match rule %s of %s: %w", candidates[i].ID, candidates[i].Principal, err)
		}
		candidates[i].Matches = len(explain) > 0
	}

	return candidates, nil
}

// candidateRule returns a stored rule as a candidate for a check, or false when its object or action can't match.
func candidateRule(principal string, paths [][]string, rule []string, object any, action string) (ExplainedRule, bool) {
	if len(rule) < 4 {
		return ExplainedRule{}, false
	}

	candidate := ExplainedRule{Principal: principal, Paths: paths, Object: rule[1], Action: rule[2], Effect: rule[3]}
	if len(rule) > 4 {
		candidate.Condition = rule[4]
	}
	if len(rule) > 5 {
		candidate.ID = rule[5]
	}

	if s, isString := object.(string); candidate.Object != "*" && (!isString || s != candidate.Object) {
		return ExplainedRule{}, false
	}
	if !matchAction(action, candidate.Action) {
		return ExplainedRule{}, false
	}

	return candidate, true
}

// rolePaths returns the subject and every role it holds, directly or through other roles, nearest first, with the chains
// of roles through which it holds each. Every role has its shortest chain; others are added up to maxRolePaths.
func rolePaths(subject string) ([]string, map[string][][]string, error) {
	rolesOf := map[string][]string{}
	roles := func(name string) ([]string, error) {
		if r, ok := rolesOf[name]; ok {
			return r, nil
		}
		r, err := enforcer.GetRolesForUser(name)
		if err != nil {
			return nil, err
		}
		slices.Sort(r)
		rolesOf[name] = r
		return r, nil
	}

	principals := []string{subject}
	paths := map[string][][]string{}
	shortest := map[string][]string{subject: nil}
	for i := 0; i < len(principals); i++ {
		current := principals[i]
		next, err := roles(current)
		if err != nil {
			return nil, nil, err
		}
		for _, role := range next {
			if _, seen := shortest[role]; seen {
				continue
			}
			shortest[role] = append(slices.Clone(shortest[current]), role)
			paths[role] = [][]string{shortest[role]}
			principals = append(principals, role)
		}
	}

	count := len(principals) - 1
	var walk func(current string, path []string) error
	walk = func(current string, path []string) error {
		next, err := roles(current)
		if err != nil {
			return err
		}
		for _, role := range next {
			if count >= maxRolePaths {
				return nil
			}
			if role == subject || slices.Contains(path, role) {
				continue
			}
			chain := append(slices.Clone(path), role)
			if !slices.ContainsFunc(paths[role], func(p []string) bool { return slices.Equal(p, chain) }) {
				paths[role] = append(paths[role], chain)
				count++
			}
			if err := walk(role, chain); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(subject, nil); err != nil {
		return nil, nil, err
	}
	return principals, paths, nil
}
