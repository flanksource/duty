package rbac

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
)

// ActionContract returns the resource types an action accepts as its primary resource and as a target.
// ok is false for an action without a contract. Mission Control sets it; without it, no request fits.
var ActionContract func(action string) (resources, targets []string, ok bool)

// ScopeRef is how a Scope appears in a request's membership lists and in Casbin conditions.
func ScopeRef(scopeID string) string {
	return "scope:" + strings.ToLower(scopeID)
}

type requestResource struct {
	kind string
	id   uuid.UUID
}

// requestResources returns the resources an authorization request involves: the ones with an id.
func requestResources(attr *models.ABACAttribute) []requestResource {
	var resources []requestResource
	add := func(kind string, id uuid.UUID) {
		if id != uuid.Nil {
			resources = append(resources, requestResource{kind: kind, id: id})
		}
	}

	add(policy.ResourceConfig, attr.Config.ID)
	add(policy.ResourceComponent, attr.Component.ID)
	add(policy.ResourceCheck, attr.Check.ID)
	add(policy.ResourceCanary, attr.Canary.ID)
	add(policy.ResourcePlaybook, attr.Playbook.ID)
	add(policy.ResourceView, attr.View.ID)
	add(policy.ResourceConnection, attr.Connection.ID)
	return resources
}

// MembershipRefs returns the resources of a request whose Scope membership is stored,
// for reading an operation's snapshot (membership.ForOperation).
func MembershipRefs(attrs ...*models.ABACAttribute) []membership.Ref {
	var refs []membership.Ref
	for _, attr := range attrs {
		if attr == nil {
			continue
		}
		for _, r := range requestResources(attr) {
			if membership.Supported(r.kind) {
				refs = append(refs, membership.Ref{Type: r.kind, ID: r.id})
			}
		}
	}
	return refs
}

// withMembership returns a copy of the request with its Membership filled in for the action.
//
// Memberships come from the context's snapshot when it covers the request (membership.ForOperation),
// so every check of an operation sees one moment. Otherwise they're read for this check alone.
// If they can't be read, the request fits nothing: it matches no allow rule and every deny rule.
func withMembership(ctx context.Context, attr *models.ABACAttribute, action string) *models.ABACAttribute {
	if attr == nil {
		return nil
	}

	out := *attr
	out.Membership = models.ABACMembership{}

	resources := requestResources(attr)
	refs := MembershipRefs(attr)

	snapshot := membership.SnapshotFrom(ctx)
	if len(refs) > 0 && !snapshot.Covers(refs...) {
		if ctx.DB() == nil {
			return &out
		}

		var err error
		if snapshot, err = membership.Read(ctx, refs...); err != nil {
			ctx.Errorf("failed to read scope membership for %s: %v", action, err)
			return &out
		}
	}

	scopesOf := func(r requestResource) []any {
		var refs []any
		for _, id := range snapshot.Scopes(membership.Ref{Type: r.kind, ID: r.id}) {
			refs = append(refs, ScopeRef(id.String()))
		}
		return refs
	}

	m := &out.Membership
	for _, r := range resources {
		scopes := scopesOf(r)
		switch r.kind {
		case policy.ResourceConfig:
			m.Config = scopes
		case policy.ResourceComponent:
			m.Component = scopes
		case policy.ResourceCheck:
			m.Check = scopes
		case policy.ResourceCanary:
			m.Canary = scopes
		case policy.ResourcePlaybook:
			m.Playbook = scopes
		case policy.ResourceConnection:
			m.Connection = scopes
		}
	}

	if ActionContract == nil {
		return &out
	}
	resourceTypes, targetTypes, ok := ActionContract(action)
	if !ok {
		return &out
	}

	var primary, targets []requestResource
	fits := true
	for _, r := range resources {
		switch {
		case slices.Contains(resourceTypes, r.kind):
			primary = append(primary, r)
		case slices.Contains(targetTypes, r.kind):
			targets = append(targets, r)
		default:
			fits = false
		}
	}

	m.HasTarget = len(targets) > 0
	m.Fits = fits && len(primary) == 1 && len(targets) <= 1
	if m.Fits {
		m.Resource = scopesOf(primary[0])
		if len(targets) == 1 {
			m.Target = scopesOf(targets[0])
		}
	}

	return &out
}

// RuleCondition returns the Casbin condition of a Role rule, over the request's Membership.
//
// An allow rule's condition holds when the request fits the action, its primary resource is in every one of
// resourceScopes, and its target is in every one of targetScopes, or there's no target when targetScopes is empty.
// A deny rule's holds when the request doesn't fit, or when the same membership test holds.
func RuleCondition(resourceScopes, targetScopes []string, deny bool) (string, error) {
	if len(resourceScopes) == 0 {
		return "", fmt.Errorf("a rule needs at least one resource scope")
	}

	test := []string{}
	if len(targetScopes) == 0 {
		test = append(test, "!r.obj.Membership.HasTarget")
	} else {
		test = append(test, "r.obj.Membership.HasTarget")
	}

	for _, list := range []struct {
		field  string
		scopes []string
	}{{"Resource", resourceScopes}, {"Target", targetScopes}} {
		scopes := slices.Clone(list.scopes)
		slices.Sort(scopes)
		for _, scope := range slices.Compact(scopes) {
			if uuid.Validate(scope) != nil {
				return "", fmt.Errorf("scope id %q isn't a uuid", scope)
			}
			test = append(test, fmt.Sprintf("'%s' in r.obj.Membership.%s", ScopeRef(scope), list.field))
		}
	}

	if deny {
		return "!r.obj.Membership.Fits || (" + strings.Join(test, " && ") + ")", nil
	}
	return "r.obj.Membership.Fits && " + strings.Join(test, " && "), nil
}

// ScopeCondition returns the Casbin condition of a Permission that names a Scope, for the Scope's resources
// of one type: the request has a resource of the type, and it's in the Scope.
func ScopeCondition(resourceType, scopeID string) (string, error) {
	if uuid.Validate(scopeID) != nil {
		return "", fmt.Errorf("scope id %q isn't a uuid", scopeID)
	}

	field := map[string]string{
		policy.ResourceConfig:     "Config",
		policy.ResourceComponent:  "Component",
		policy.ResourceCheck:      "Check",
		policy.ResourceCanary:     "Canary",
		policy.ResourcePlaybook:   "Playbook",
		policy.ResourceConnection: "Connection",
	}[resourceType]
	if field == "" {
		return "", fmt.Errorf("membership of %s isn't stored", resourceType)
	}

	return fmt.Sprintf("'%s' in r.obj.Membership.%s", ScopeRef(scopeID), field), nil
}
