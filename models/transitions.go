package models

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/oops"
	"gorm.io/gorm"
)

// ErrTransition is matched by every *TransitionError.
var ErrTransition = errors.New("invalid status transition")

// Deprecated: use ErrTransition.
var ErrPlaybookTransition = ErrTransition

// TransitionError is returned when a row isn't in a status that may move to the requested one,
// usually because another writer moved it first.
type TransitionError struct {
	Entity string
	ID     uuid.UUID

	// From is the status the row was found in. It is empty when the row doesn't exist or has no status.
	From string
	To   string
}

// Deprecated: use TransitionError.
type PlaybookTransitionError = TransitionError

func (e *TransitionError) Error() string {
	if e.From == "" {
		return fmt.Sprintf("%s %s not found: cannot move it to %q", e.Entity, e.ID, e.To)
	}
	return fmt.Sprintf("%s %s cannot move from %q to %q", e.Entity, e.ID, e.From, e.To)
}

func (e *TransitionError) Is(target error) bool {
	return target == ErrTransition
}

// sourcesOf returns every status that may move to `to` according to the transitions table.
func sourcesOf[S ~string](transitions map[S][]S, to S) []S {
	var from []S
	for s, next := range transitions {
		if slices.Contains(next, to) {
			from = append(from, s)
		}
	}
	slices.Sort(from)
	return from
}

// transition moves the row to `to` and writes the extra columns, in a single UPDATE that only matches
// when the row's current status may move to `to`. So the check and the write can't be interleaved
// with another writer.
func transition[S ~string](db *gorm.DB, model any, entity string, id uuid.UUID, transitions map[S][]S, to S, columns map[string]any) error {
	if err := rejectStatusColumn(columns); err != nil {
		return err
	}

	updates := map[string]any{}
	for k, v := range columns {
		updates[k] = v
	}
	updates["status"] = to

	res := db.Model(model).Where("id = ?", id).Where("status IN ?", sourcesOf(transitions, to)).UpdateColumns(updates)
	if res.Error != nil {
		return oops.Tags("db").Wrap(res.Error)
	}
	if res.RowsAffected == 1 {
		return nil
	}

	var current []sql.NullString
	if err := db.Model(model).Where("id = ?", id).Pluck("status", &current).Error; err != nil {
		return oops.Tags("db").Wrap(err)
	}

	terr := &TransitionError{Entity: entity, ID: id, To: string(to)}
	if len(current) > 0 {
		terr.From = current[0].String
	}
	return terr
}

func rejectStatusColumn(columns map[string]any) error {
	// GORM resolves a key by column name or by Go field name, so "Status" also writes the column.
	for k := range columns {
		if strings.EqualFold(k, "status") {
			return fmt.Errorf("status can only be changed through a transition method")
		}
	}
	return nil
}
