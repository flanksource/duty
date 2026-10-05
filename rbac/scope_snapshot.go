package rbac

import (
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/flanksource/duty/context"
)

// ResourceRef identifies a resource.
type ResourceRef struct {
	Type string
	ID   uuid.UUID
}

// ScopeSnapshot holds the Scopes each resource is in, read from scope_members at one moment.
// A resource is in a Scope when the Scope has a membership row for it, or one for its whole type.
type ScopeSnapshot struct {
	scopes map[ResourceRef][]uuid.UUID
}

// Covers reports whether the snapshot was read for every resource.
func (s *ScopeSnapshot) Covers(refs ...ResourceRef) bool {
	if s == nil {
		return false
	}
	for _, ref := range refs {
		if _, ok := s.scopes[ref]; !ok {
			return false
		}
	}
	return true
}

// Scopes returns the Scopes the resource is in, sorted.
func (s *ScopeSnapshot) Scopes(ref ResourceRef) []uuid.UUID {
	if s == nil {
		return nil
	}
	return s.scopes[ref]
}

// NewScopeSnapshot returns a snapshot holding the given memberships. For tests.
func NewScopeSnapshot(scopes map[ResourceRef][]uuid.UUID) *ScopeSnapshot {
	s := &ScopeSnapshot{scopes: map[ResourceRef][]uuid.UUID{}}
	for ref, ids := range scopes {
		ids = slices.Clone(ids)
		slices.SortFunc(ids, func(a, b uuid.UUID) int { return compareUUID(a, b) })
		s.scopes[ref] = ids
	}
	return s
}

// ReadScopeSnapshot reads the Scopes every resource is in, in one query, so they're all of one moment.
func ReadScopeSnapshot(ctx context.Context, refs ...ResourceRef) (*ScopeSnapshot, error) {
	snapshot := &ScopeSnapshot{scopes: map[ResourceRef][]uuid.UUID{}}
	var kinds, ids []string
	for _, ref := range refs {
		if ref.ID == uuid.Nil {
			continue
		}
		snapshot.scopes[ref] = nil
		kinds = append(kinds, ref.Type)
		ids = append(ids, ref.ID.String())
	}
	if len(kinds) == 0 {
		return snapshot, nil
	}

	var rows []struct {
		ResourceType string
		ResourceID   uuid.UUID
		ScopeID      uuid.UUID
	}
	err := ctx.DB().Raw(`WITH refs AS (
			SELECT * FROM unnest(?::text[], ?::uuid[]) AS r(resource_type, resource_id)
		)
		SELECT DISTINCT r.resource_type, r.resource_id, m.scope_id
		FROM refs r
		JOIN scope_members m ON m.resource_type = r.resource_type AND (m.resource_id = r.resource_id OR m.resource_id IS NULL)`,
		pq.StringArray(kinds), pq.StringArray(ids)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		ref := ResourceRef{Type: row.ResourceType, ID: row.ResourceID}
		snapshot.scopes[ref] = append(snapshot.scopes[ref], row.ScopeID)
	}
	for ref, scopes := range snapshot.scopes {
		slices.SortFunc(scopes, compareUUID)
		snapshot.scopes[ref] = slices.Compact(scopes)
	}

	return snapshot, nil
}

type scopeSnapshotKey struct{}

// WithScopeSnapshot returns a context whose checks use the snapshot for the resources it covers,
// so all checks of one operation see the membership of one moment.
func WithScopeSnapshot(ctx context.Context, snapshot *ScopeSnapshot) context.Context {
	return ctx.WithValue(scopeSnapshotKey{}, snapshot)
}

// ScopeSnapshotFrom returns the snapshot of the context, if any.
func ScopeSnapshotFrom(ctx context.Context) *ScopeSnapshot {
	if v, ok := ctx.Value(scopeSnapshotKey{}).(*ScopeSnapshot); ok {
		return v
	}
	return nil
}

// WithOperation reads the Scopes of every resource an operation involves, and returns a context whose
// checks use them. Call it once, before an operation's checks.
func WithOperation(ctx context.Context, refs ...ResourceRef) (context.Context, error) {
	snapshot, err := ReadScopeSnapshot(ctx, refs...)
	if err != nil {
		return ctx, err
	}
	return WithScopeSnapshot(ctx, snapshot), nil
}

func compareUUID(a, b uuid.UUID) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}
