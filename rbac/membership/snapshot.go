package membership

// One user action can need several permission checks. Each check asks: which Scopes is this resource in?
//
// If each check asked the database separately, the answers could change between checks. Someone might edit a
// Scope in the middle. Then one check would use the old Scope and the next check would use the new one.
//
// Example: Alice may run playbooks on anything in the Scope "production". She runs the playbook
// "restart-deployment" on the Kubernetes Deployment "payments-api". Two checks run:
//
//  1. Is "restart-deployment" in a Scope Alice can use? The database says yes, it is in "production".
//  2. Is "payments-api" in a Scope Alice can use?
//
// Between check 1 and check 2, an admin removes "payments-api" from "production". Check 2 now says no. Alice's
// request was judged against two different versions of "production".
//
// A Snapshot prevents this. It reads the Scopes of all the resources at once, in one query, before the checks
// start. The checks then read from the Snapshot, not the database. So every check gets the same answer.
//
// How to use it:
//   - Call ForOperation before the checks. It reads the Snapshot and stores it in the context.
//   - Call SnapshotFrom inside a check to get the Snapshot back.
//   - Call Covers to see if the Snapshot has the resource. If it doesn't, the check must ask the database itself.

import (
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/flanksource/duty/context"
)

// Ref identifies a resource.
type Ref struct {
	Type string
	ID   uuid.UUID
}

// Snapshot holds the Scopes each resource is in, read from scope_members at one moment.
// A resource is in a Scope when the Scope has a membership row for it, or one for its whole type.
type Snapshot struct {
	scopes map[Ref][]uuid.UUID
}

// Covers reports whether the snapshot was read for every resource.
func (s *Snapshot) Covers(refs ...Ref) bool {
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
func (s *Snapshot) Scopes(ref Ref) []uuid.UUID {
	if s == nil {
		return nil
	}
	return s.scopes[ref]
}

// NewSnapshot returns a snapshot holding the given memberships. For tests.
func NewSnapshot(scopes map[Ref][]uuid.UUID) *Snapshot {
	s := &Snapshot{scopes: map[Ref][]uuid.UUID{}}
	for ref, ids := range scopes {
		ids = slices.Clone(ids)
		slices.SortFunc(ids, func(a, b uuid.UUID) int { return compareUUID(a, b) })
		s.scopes[ref] = ids
	}
	return s
}

// Read reads the Scopes every resource is in, in one query, so they're all of one moment.
func Read(ctx context.Context, refs ...Ref) (*Snapshot, error) {
	snapshot := &Snapshot{scopes: map[Ref][]uuid.UUID{}}
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
		ref := Ref{Type: row.ResourceType, ID: row.ResourceID}
		snapshot.scopes[ref] = append(snapshot.scopes[ref], row.ScopeID)
	}
	for ref, scopes := range snapshot.scopes {
		slices.SortFunc(scopes, compareUUID)
		snapshot.scopes[ref] = slices.Compact(scopes)
	}

	return snapshot, nil
}

type snapshotKey struct{}

// WithSnapshot returns a context whose checks use the snapshot for the resources it covers,
// so all checks of one operation see the membership of one moment.
func WithSnapshot(ctx context.Context, snapshot *Snapshot) context.Context {
	return ctx.WithValue(snapshotKey{}, snapshot)
}

// SnapshotFrom returns the snapshot of the context, if any.
func SnapshotFrom(ctx context.Context) *Snapshot {
	if v, ok := ctx.Value(snapshotKey{}).(*Snapshot); ok {
		return v
	}
	return nil
}

// ForOperation reads the memberships of every resource an operation involves, and returns a context whose
// checks use them. Call it once, before an operation's checks.
func ForOperation(ctx context.Context, refs ...Ref) (context.Context, error) {
	snapshot, err := Read(ctx, refs...)
	if err != nil {
		return ctx, err
	}
	return WithSnapshot(ctx, snapshot), nil
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
