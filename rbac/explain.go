package rbac

import (
	"errors"
	"fmt"
	"slices"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/policy"
)

// maxRoleDepth is how many grouping edges a subject's roles may be away, as for casbin's default role manager.
const maxRoleDepth = 10

// ExplainedRule is a rule of the policy that a check could match.
type ExplainedRule struct {
	// Principal is who the rule is filed under: the subject itself, or a role the subject holds.
	Principal string `json:"principal"`

	// Paths are the shortest chains of roles from the subject to Principal, one through each principal that holds
	// Principal directly, each with Principal last. It's empty when the rule is filed under the subject itself.
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

// Explanation is how the policy decides a check.
type Explanation struct {
	// Allowed is the answer: some matching rule allows the check, and none denies it.
	Allowed bool

	// Rules are every rule the subject holds, directly or through its roles, for the action and object.
	Rules []ExplainedRule
}

// PolicySnapshot is the policy of the enforcer at one moment, so the checks explained with it agree with each other.
type PolicySnapshot struct {
	model    model.Model
	roles    map[string][]string
	policies map[string][][]string
}

// SnapshotPolicy copies the enforcer's policy and roles.
func SnapshotPolicy() (*PolicySnapshot, error) {
	if enforcer == nil {
		return nil, errors.New("rbac isn't initialized")
	}

	m, err := model.NewModelFromString(DefaultModel)
	if err != nil {
		return nil, err
	}
	snapshot := &PolicySnapshot{model: m, roles: map[string][]string{}, policies: map[string][][]string{}}

	// Read under the enforcer's lock, which a reload takes to swap the policy in
	lock := enforcer.GetLock()
	lock.RLock()
	defer lock.RUnlock()

	current := enforcer.GetModel()
	if grouping, ok := current["g"]["g"]; ok {
		for _, rule := range grouping.Policy {
			if len(rule) >= 2 {
				snapshot.roles[rule[0]] = append(snapshot.roles[rule[0]], rule[1])
			}
		}
	}
	if policies, ok := current["p"]["p"]; ok {
		for _, rule := range policies.Policy {
			if len(rule) >= 4 {
				snapshot.policies[rule[0]] = append(snapshot.policies[rule[0]], slices.Clone(rule))
			}
		}
	}
	return snapshot, nil
}

// Principals returns every principal the policy holds a rule or a role for, sorted.
func (s *PolicySnapshot) Principals() []string {
	principals := append(lo.Keys(s.roles), lo.Keys(s.policies)...)
	slices.Sort(principals)
	return slices.Compact(principals)
}

// Explain decides a check of the action by the subject on the object, as HasPermission and Check decide it, and returns
// every rule that could decide it. Like Check, a check on an RBAC object gives a subject with implicit grants
// "everyone"; like HasPermission, a check on resources doesn't.
//
// object is an RBAC object, e.g. "catalog", or a *models.ABACAttribute, whose Scope membership is read for the
// action as HasPermission reads it.
func (s *PolicySnapshot) Explain(ctx context.Context, subject string, object any, action string) (Explanation, error) {
	switch o := object.(type) {
	case string:
	case *models.ABACAttribute:
		object = withMembership(ctx, o, action)
	default:
		return Explanation{}, fmt.Errorf("unsupported object %T", object)
	}

	_, implicit := object.(string)
	principals, paths := s.rolePaths(subject, implicit && HasImplicitGrants(subject))

	var explanation Explanation
	for _, principal := range principals {
		for _, rule := range s.policies[principal] {
			if candidate, ok := candidateRule(principal, paths[principal], rule, object, action); ok {
				explanation.Rules = append(explanation.Rules, candidate)
			}
		}
	}

	// Each candidate is matched alone, filed as an allow, so the check is allowed exactly when that rule matches,
	// whatever its effect. The enforcer holds one rule at a time: casbin evaluates every rule it holds on each check.
	isolated, err := casbin.NewEnforcer(s.model.Copy())
	if err != nil {
		return Explanation{}, err
	}
	AddCustomFunctions(isolated)
	for i, rule := range explanation.Rules {
		isolated.ClearPolicy()
		if _, err := isolated.AddPolicy("rule", rule.Object, rule.Action, "allow", rule.Condition, rule.ID); err != nil {
			return Explanation{}, err
		}
		matches, err := isolated.Enforce("rule", object, action)
		if err != nil {
			return Explanation{}, fmt.Errorf("failed to match rule %s of %s: %w", rule.ID, rule.Principal, err)
		}
		explanation.Rules[i].Matches = matches
	}

	matching := func(effect string) bool {
		return lo.SomeBy(explanation.Rules, func(r ExplainedRule) bool { return r.Matches && r.Effect == effect })
	}
	explanation.Allowed = matching("allow") && !matching("deny")
	return explanation, nil
}

// candidateRule returns a stored rule as a candidate for a check, or false when its object or action can't match.
func candidateRule(principal string, paths [][]string, rule []string, object any, action string) (ExplainedRule, bool) {
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

// rolePaths returns the subject and every role it holds within maxRoleDepth, nearest first, with the shortest chain of
// roles to each role through each principal that holds it directly. With everyone set, the subject also holds everyone.
func (s *PolicySnapshot) rolePaths(subject string, everyone bool) ([]string, map[string][][]string) {
	rolesOf := func(name string) []string {
		roles := slices.Clone(s.roles[name])
		if name == subject && everyone && !slices.Contains(roles, policy.RoleEveryone) {
			roles = append(roles, policy.RoleEveryone)
		}
		slices.Sort(roles)
		return slices.Compact(roles)
	}

	principals := []string{subject}
	shortest := map[string][]string{subject: nil}
	holders := map[string][]string{}
	for depth, level := 0, []string{subject}; depth < maxRoleDepth && len(level) > 0; depth++ {
		var next []string
		for _, current := range level {
			for _, role := range rolesOf(current) {
				if role == subject {
					continue
				}
				if _, seen := shortest[role]; !seen {
					shortest[role] = append(slices.Clone(shortest[current]), role)
					principals = append(principals, role)
					next = append(next, role)
				}
				holders[role] = append(holders[role], current)
			}
		}
		level = next
	}

	paths := map[string][][]string{}
	for role, through := range holders {
		for _, holder := range through {
			if slices.Contains(shortest[holder], role) {
				continue
			}
			path := append(slices.Clone(shortest[holder]), role)
			if !slices.ContainsFunc(paths[role], func(p []string) bool { return slices.Equal(p, path) }) {
				paths[role] = append(paths[role], path)
			}
		}
	}
	return principals, paths
}
