package models

import (
	"slices"
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

// IsFinal reports whether a playbook run in this status can never change again.
func (s PlaybookRunStatus) IsFinal() bool {
	return slices.Contains(PlaybookRunStatusFinalStates, s)
}
