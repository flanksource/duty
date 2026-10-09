package rbac

import (
	"slices"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

type scopeLimitsKey struct{}

// LimitToScopes returns a context whose resource checks (HasPermission) allow only resources in at least one of the
// Scopes, on top of what the subject's rules allow. It's the check side of rls.Grants.Limit, which limits listings.
// Limits add up: a context limited twice allows only resources in a Scope of each. A limit naming no Scope allows
// no resource.
func LimitToScopes(ctx context.Context, scopeIDs []string) context.Context {
	limit := slices.Clone(scopeIDs)
	if limit == nil {
		limit = []string{}
	}
	return ctx.WithValue(scopeLimitsKey{}, append(slices.Clone(ScopeLimits(ctx)), limit))
}

// ScopeLimits returns the limits of the context, oldest first. Nil when it isn't limited.
func ScopeLimits(ctx context.Context) [][]string {
	limits, _ := ctx.Value(scopeLimitsKey{}).([][]string)
	return limits
}

// withoutScopeLimits returns a context whose resource checks aren't limited, e.g. to check what another subject
// may do: a requester's limit is about the requester's request, not about other subjects' access.
func withoutScopeLimits(ctx context.Context) context.Context {
	if ScopeLimits(ctx) == nil {
		return ctx
	}
	return ctx.WithValue(scopeLimitsKey{}, nil)
}

// withinScopeLimits reports whether every resource of the request, with its Membership filled in, is in at least
// one Scope of each of the context's limits. Resources whose membership isn't stored, e.g. views, aren't limited,
// as their listings aren't.
func withinScopeLimits(ctx context.Context, attr *models.ABACAttribute) bool {
	limits := ScopeLimits(ctx)
	if len(limits) == 0 || attr == nil {
		return true
	}

	for _, r := range requestResources(attr) {
		in := membershipOf(&attr.Membership, r.kind)
		if in == nil {
			continue
		}
		for _, limit := range limits {
			if !slices.ContainsFunc(limit, func(id string) bool { return slices.Contains(*in, any(ScopeRef(id))) }) {
				return false
			}
		}
	}
	return true
}
