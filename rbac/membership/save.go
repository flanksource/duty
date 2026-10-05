package membership

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
)

var (
	// LockTimeout is how long saving a Scope waits for the membership lock before backing off.
	// Writers already holding it shared, e.g. a long scraper transaction, would otherwise stall every resource write
	// queued behind the exclusive request.
	LockTimeout = time.Second

	// LockRetries is how many times saving a Scope retries after LockTimeout, backing off between attempts.
	LockRetries = 5

	// LockBackoff is the wait before the first retry. It doubles after each one.
	LockBackoff = 200 * time.Millisecond
)

// ErrLockTimeout is returned when the membership lock couldn't be taken within LockRetries.
var ErrLockTimeout = errors.New("timed out waiting for the scope membership lock")

// Rebuild rewrites the Scope's stored targets and rebuilds its members, in the context's transaction. It doesn't save
// the Scope itself: the caller saves the Scope row in the same transaction, and readers see the old membership until
// it commits and the new one after. resolved are the Scope's validated targets with their agents resolved to ids.
// Targets of types whose membership isn't stored, e.g. views, are ignored, and a whole-type target is stored as one
// member with no resource. Nothing is written when the targets haven't changed, and a rebuild writes only the members
// that change. It reports whether anything was rebuilt.
func Rebuild(ctx context.Context, scopeID uuid.UUID, resolved []Target) (bool, error) {
	var rows []targetRow
	var whole []string
	for _, t := range resolved {
		if !Supported(t.Type) {
			continue
		} else if t.WholeType() {
			whole = append(whole, t.Type)
			continue
		}
		row, err := t.row(scopeID)
		if err != nil {
			return false, fmt.Errorf("scope %s: %s target: %w", scopeID, t.Type, err)
		}
		rows = append(rows, row)
	}
	slices.Sort(whole)
	whole = slices.Compact(whole)

	// Targets of a type the Scope selects entirely add nothing
	rows = slices.DeleteFunc(rows, func(r targetRow) bool { return slices.Contains(whole, r.ResourceType) })

	unchanged, err := stored(ctx.DB(), scopeID, rows, whole)
	if err != nil {
		return false, err
	} else if unchanged {
		return false, nil
	}

	err = withLock(ctx, func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM scope_targets WHERE scope_id = ?", scopeID).Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			if err := tx.Table("scope_targets").Omit("lookup_keys").Create(&rows).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM scope_members WHERE scope_id = ? AND resource_id IS NULL AND NOT (resource_type = ANY (?::text[]))",
			scopeID, textArray(whole)).Error; err != nil {
			return err
		}
		for _, kind := range whole {
			if err := tx.Exec(`DELETE FROM scope_members WHERE scope_id = ? AND resource_type = ? AND resource_id IS NOT NULL`, scopeID, kind).Error; err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO scope_members (scope_id, resource_type) VALUES (?, ?)
				ON CONFLICT (scope_id, resource_type) WHERE resource_id IS NULL DO NOTHING`, scopeID, kind).Error; err != nil {
				return err
			}
		}

		return tx.Exec("SELECT scope_rebuild_members(?)", scopeID).Error
	})
	if err != nil {
		return false, fmt.Errorf("failed to rebuild the membership of scope %s: %w", scopeID, err)
	}
	return true, nil
}

// Clear deletes the Scope's stored targets and members in the context's transaction, e.g. when it becomes invalid or
// is deleted. It doesn't delete the Scope row. The Scope admits nothing from then on.
func Clear(ctx context.Context, scopeID uuid.UUID) error {
	if built, err := Built(ctx, scopeID); err != nil || !built {
		return err
	}

	err := withLock(ctx, func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM scope_targets WHERE scope_id = ?", scopeID).Error; err != nil {
			return err
		}
		return tx.Exec("DELETE FROM scope_members WHERE scope_id = ?", scopeID).Error
	})
	if err != nil {
		return fmt.Errorf("failed to delete the membership of scope %s: %w", scopeID, err)
	}
	return nil
}

// Built reports whether the Scope has stored targets or members, i.e. whether its membership has been built.
func Built(ctx context.Context, scopeID uuid.UUID) (bool, error) {
	var has bool
	err := ctx.DB().Raw(`SELECT EXISTS (SELECT 1 FROM scope_targets WHERE scope_id = ?)
		OR EXISTS (SELECT 1 FROM scope_members WHERE scope_id = ?)`, scopeID, scopeID).Scan(&has).Error
	return has, err
}

// stored reports whether the Scope's stored targets and whole types are exactly the given ones.
func stored(db *gorm.DB, scopeID uuid.UUID, rows []targetRow, whole []string) (bool, error) {
	var current []targetRow
	if err := db.Raw(`SELECT scope_id, resource_type, resource_id::text, name, name_prefix, namespace, agent_id::text, types::text, tags::text, labels::text
		FROM scope_targets WHERE scope_id = ?`, scopeID).Scan(&current).Error; err != nil {
		return false, err
	}

	var currentWhole []string
	if err := db.Raw("SELECT resource_type FROM scope_members WHERE scope_id = ? AND resource_id IS NULL ORDER BY resource_type", scopeID).
		Scan(&currentWhole).Error; err != nil {
		return false, err
	}

	if len(current) == 0 && len(currentWhole) == 0 {
		// Never built, or deleted: a Scope that selects nothing storable still has nothing to build
		return len(rows) == 0 && len(whole) == 0, nil
	}

	keys := func(rows []targetRow) []string {
		var out []string
		for _, r := range rows {
			out = append(out, r.normalized().key())
		}
		slices.Sort(out)
		return out
	}

	return slices.Equal(keys(current), keys(rows)) && slices.Equal(currentWhole, whole), nil
}

// normalized returns the row in one form, so a row read back from Postgres and a new one compare equal.
func (r targetRow) normalized() targetRow {
	for _, field := range []**string{&r.Tags, &r.Labels} {
		if *field != nil {
			var pairs map[string]string
			if err := json.Unmarshal([]byte(**field), &pairs); err == nil {
				raw, _ := json.Marshal(pairs)
				s := string(raw)
				*field = &s
			}
		}
	}
	if r.Types != nil {
		var values pq.StringArray
		if err := values.Scan(*r.Types); err == nil {
			s := textArray(values)
			r.Types = &s
		}
	}
	return r
}

// textArray returns the values as a Postgres text array literal.
func textArray(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	value, _ := pq.StringArray(values).Value()
	return value.(string)
}

// withLock runs fn holding the membership lock exclusively, so no resource is written while a Scope's targets change.
// It waits for the lock at most LockTimeout, then backs off and retries, LockRetries times.
// Each attempt runs in a savepoint of the context's transaction.
func withLock(ctx context.Context, fn func(tx *gorm.DB) error) error {
	backoff := LockBackoff
	for attempt := 0; ; attempt++ {
		err := ctx.DB().Transaction(func(tx *gorm.DB) error {
			var previous string
			if err := tx.Raw("SELECT current_setting('lock_timeout')").Scan(&previous).Error; err != nil {
				return err
			}
			if err := tx.Exec("SELECT set_config('lock_timeout', ?, true)", fmt.Sprintf("%dms", LockTimeout.Milliseconds())).Error; err != nil {
				return err
			}
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('scope_membership'))").Error; err != nil {
				return err
			}
			if err := tx.Exec("SELECT set_config('lock_timeout', ?, true)", previous).Error; err != nil {
				return err
			}
			return fn(tx)
		})

		var pgErr *pgconn.PgError
		if err == nil || !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
			return err
		} else if attempt >= LockRetries {
			return ErrLockTimeout
		}

		ctx.Debugf("scope membership lock busy, retrying in %s", backoff)
		time.Sleep(backoff)
		backoff *= 2
	}
}
