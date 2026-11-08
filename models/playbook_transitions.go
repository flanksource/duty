package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/oops"
	"gorm.io/gorm"
)

// playbookRunTransitions lists, for each status of a playbook run, the statuses it may move to.
// A status that isn't a key is final: the run can never leave it.
var playbookRunTransitions = map[PlaybookRunStatus][]PlaybookRunStatus{
	PlaybookRunStatusPendingApproval: {
		PlaybookRunStatusScheduled,
		PlaybookRunStatusCancelled, PlaybookRunStatusTimedOut,
	},
	PlaybookRunStatusScheduled: {
		PlaybookRunStatusRunning, PlaybookRunStatusWaiting, PlaybookRunStatusSleeping, PlaybookRunStatusRetrying,
		PlaybookRunStatusCompleted, PlaybookRunStatusFailed, PlaybookRunStatusCancelled, PlaybookRunStatusTimedOut,
	},
	PlaybookRunStatusSleeping: {
		PlaybookRunStatusRunning, PlaybookRunStatusWaiting, PlaybookRunStatusRetrying,
		PlaybookRunStatusCompleted, PlaybookRunStatusFailed, PlaybookRunStatusCancelled, PlaybookRunStatusTimedOut,
	},
	PlaybookRunStatusRetrying: {
		PlaybookRunStatusRunning, PlaybookRunStatusWaiting, PlaybookRunStatusSleeping,
		PlaybookRunStatusCompleted, PlaybookRunStatusFailed, PlaybookRunStatusCancelled, PlaybookRunStatusTimedOut,
	},
	PlaybookRunStatusRunning: {
		PlaybookRunStatusScheduled,
		PlaybookRunStatusFailed, PlaybookRunStatusCancelled, PlaybookRunStatusTimedOut,
	},
	PlaybookRunStatusWaiting: {
		PlaybookRunStatusScheduled,
		PlaybookRunStatusFailed, PlaybookRunStatusCancelled, PlaybookRunStatusTimedOut,
	},
}

// playbookActionTransitions lists, for each status of a playbook run action, the statuses it may move to.
// A status that isn't a key is final: the action can never leave it.
var playbookActionTransitions = map[PlaybookActionStatus][]PlaybookActionStatus{
	PlaybookActionStatusScheduled: {
		PlaybookActionStatusRunning, PlaybookActionStatusFailed, PlaybookActionStatusSkipped,
	},
	PlaybookActionStatusWaiting: {
		PlaybookActionStatusRunning, PlaybookActionStatusFailed, PlaybookActionStatusSkipped,
	},
	PlaybookActionStatusRunning: {
		PlaybookActionStatusCompleted, PlaybookActionStatusFailed, PlaybookActionStatusSkipped,
		PlaybookActionStatusWaitingChildren,
	},
	PlaybookActionStatusWaitingChildren: {
		PlaybookActionStatusScheduled,
	},
}

// ErrPlaybookTransition is matched by every *PlaybookTransitionError.
var ErrPlaybookTransition = errors.New("invalid playbook status transition")

// PlaybookTransitionError is returned when a playbook run or action isn't in a status
// that may move to the requested one, usually because another writer moved it first.
type PlaybookTransitionError struct {
	Entity string
	ID     uuid.UUID

	// From is the status the row was found in. It is empty when the row doesn't exist.
	From string
	To   string
}

func (e *PlaybookTransitionError) Error() string {
	if e.From == "" {
		return fmt.Sprintf("%s %s not found: cannot move it to %q", e.Entity, e.ID, e.To)
	}
	return fmt.Sprintf("%s %s cannot move from %q to %q", e.Entity, e.ID, e.From, e.To)
}

func (e *PlaybookTransitionError) Is(target error) bool {
	return target == ErrPlaybookTransition
}

// IsFinal reports whether a playbook run in this status can never change again.
func (s PlaybookRunStatus) IsFinal() bool {
	return slices.Contains(PlaybookRunStatusFinalStates, s)
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

	var current []string
	if err := db.Model(model).Where("id = ?", id).Pluck("status", &current).Error; err != nil {
		return oops.Tags("db").Wrap(err)
	}

	terr := &PlaybookTransitionError{Entity: entity, ID: id, To: string(to)}
	if len(current) > 0 {
		terr.From = current[0]
	}
	return terr
}

func rejectStatusColumn(columns map[string]any) error {
	// GORM resolves a key by column name or by Go field name, so "Status" also writes the column.
	for k := range columns {
		if strings.EqualFold(k, "status") {
			return fmt.Errorf("status can only be changed through a playbook transition method")
		}
	}
	return nil
}
