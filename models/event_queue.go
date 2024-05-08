package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/flanksource/postq"
	"github.com/google/uuid"
)

type Event struct {
	ID          uuid.UUID           `gorm:"default:generate_ulid()"`
	Name        string              `json:"name"`
	CreatedAt   time.Time           `json:"created_at"`
	Properties  types.JSONStringMap `json:"properties"`
	Error       *string             `json:"error,omitempty"`
	Attempts    int                 `json:"attempts"`
	LastAttempt *time.Time          `json:"last_attempt"`
	Priority    int                 `json:"priority"`
}

func (t Event) ToPostQEvent() postq.Event {
	return postq.Event{
		ID:          t.ID,
		Name:        t.Name,
		Error:       t.Error,
		Attempts:    t.Attempts,
		LastAttempt: t.LastAttempt,
		Properties:  t.Properties,
		CreatedAt:   t.CreatedAt,
	}
}

// We are using the term `Event` as it represents an event in the
// event_queue table, but the table is named event_queue
// to signify it's usage as a queue
func (Event) TableName() string {
	return "event_queue"
}

type Events []Event

func (events Events) ToPostQEvents() postq.Events {
	var output []postq.Event
	for _, event := range events {
		output = append(output, event.ToPostQEvent())
	}

	return output
}

type EventQueueSummary struct {
	Name          string     `json:"name"`
	Pending       int64      `json:"pending"`
	Failed        int64      `json:"failed"`
	AvgAttempts   int64      `json:"average_attempts"`
	FirstFailure  *time.Time `json:"first_failure,omitempty"`
	LastFailure   *time.Time `json:"last_failure,omitempty"`
	MostCommonErr string     `json:"most_common_error,omitempty"`
}

func (t *EventQueueSummary) TableName() string {
	return "event_queue_summary"
}
