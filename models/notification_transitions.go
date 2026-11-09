package models

import (
	"github.com/samber/oops"
	"gorm.io/gorm"
)

// notificationSendHistoryTransitions lists, for each status of a notification send history, the statuses it may move to.
// A status that isn't a key is final: the send history can never leave it.
//
// Rows are created in sending, pending, attempting_fallback, or directly in one of the final blocked statuses
// (silenced, repeat-interval, inhibited).
var notificationSendHistoryTransitions = map[string][]string{
	NotificationStatusSending: {
		NotificationStatusSent, NotificationStatusError, NotificationStatusSkipped,
		NotificationStatusPendingPlaybookRun, NotificationStatusPendingPlaybookCompletion,
	},
	NotificationStatusPending: {
		NotificationStatusEvaluatingWaitFor,
		NotificationStatusSkipped, NotificationStatusSilenced, NotificationStatusRepeatInterval, NotificationStatusInhibited,
		NotificationStatusSent, NotificationStatusError,
		NotificationStatusPendingPlaybookRun, NotificationStatusPendingPlaybookCompletion,
	},
	NotificationStatusEvaluatingWaitFor: {
		NotificationStatusSkipped, NotificationStatusSilenced, NotificationStatusRepeatInterval, NotificationStatusInhibited,
		NotificationStatusSent, NotificationStatusError,
		NotificationStatusPendingPlaybookRun, NotificationStatusPendingPlaybookCompletion,
	},
	NotificationStatusAttemptingFallback: {
		NotificationStatusSent, NotificationStatusError, NotificationStatusSkipped,
		NotificationStatusPendingPlaybookRun, NotificationStatusPendingPlaybookCompletion,
	},

	// The playbook run can be created, and can even end, before the dispatch that triggered it records pending_playbook_run.
	// That's why every status above may also move to pending_playbook_completion.
	NotificationStatusPendingPlaybookRun: {
		NotificationStatusPendingPlaybookCompletion, NotificationStatusSent, NotificationStatusError,
	},
	NotificationStatusPendingPlaybookCompletion: {
		NotificationStatusSent, NotificationStatusError,
	},

	// A notification with onResolved retries a failed send in the same row, which has a deterministic ID.
	NotificationStatusError: {
		NotificationStatusSending,
	},
}

// Update writes columns other than status. Use Transition to change the status.
func (t NotificationSendHistory) Update(db *gorm.DB, columns map[string]any) error {
	if err := rejectStatusColumn(columns); err != nil {
		return err
	}
	return oops.Tags("db").Wrap(db.Model(&NotificationSendHistory{}).Where("id = ?", t.ID).UpdateColumns(columns).Error)
}

// Transition moves the send history to the given status and writes the extra columns,
// but only if its current status allows it. Otherwise it returns a *TransitionError.
func (t NotificationSendHistory) Transition(db *gorm.DB, to string, columns map[string]any) error {
	return transition(db, &NotificationSendHistory{}, "notification send history", t.ID, notificationSendHistoryTransitions, to, columns)
}
