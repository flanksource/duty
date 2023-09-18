package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Notification represents the notifications table
type Notification struct {
	ID             uuid.UUID           `json:"id"`
	Events         pq.StringArray      `json:"events" gorm:"type:[]text"`
	Title          string              `json:"title,omitempty"`
	Template       string              `json:"template,omitempty"`
	Filter         string              `json:"filter,omitempty"`
	PersonID       *uuid.UUID          `json:"person_id,omitempty"`
	TeamID         *uuid.UUID          `json:"team_id,omitempty"`
	Properties     types.JSONStringMap `json:"properties,omitempty"`
	CustomServices types.JSON          `json:"custom_services,omitempty" gorm:"column:custom_services"`
	CreatedBy      *uuid.UUID          `json:"created_by,omitempty"`
	UpdatedAt      time.Time           `json:"updated_at" time_format:"postgres_timestamp" gorm:"<-:false"`
	CreatedAt      time.Time           `json:"created_at" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt      *time.Time          `json:"deleted_at,omitempty"`
}

func (n *Notification) HasRecipients() bool {
	return n.TeamID != nil || n.PersonID != nil || len(n.CustomServices) != 0
}

func (n Notification) AsMap(removeFields ...string) map[string]any {
	return asMap(n, removeFields...)
}

type NotificationSendHistory struct {
	ID             uuid.UUID `json:"id,omitempty" gorm:"default:generate_ulid()"`
	NotificationID uuid.UUID `json:"notification_id"`
	Body           string    `json:"body,omitempty"`
	Error          *string   `json:"error,omitempty"`
	DurationMs     int64     `json:"duration_ms,omitempty" gorm:"column:duration_millis"`
	CreatedAt      time.Time `json:"created_at" time_format:"postgres_timestamp"`

	// Name of the original event that caused this notification
	SourceEvent string `json:"source_event"`

	// ID of the resource this notification is for
	ResourceID uuid.UUID `json:"resource_id"`

	// ID of the person this notification is for.
	PersonID *uuid.UUID `json:"person_id"`

	timeStart time.Time
}

func (n NotificationSendHistory) AsMap(removeFields ...string) map[string]any {
	return asMap(n, removeFields...)
}

func (t *NotificationSendHistory) TableName() string {
	return "notification_send_history"
}

func NewNotificationSendHistory(notificationID uuid.UUID) *NotificationSendHistory {
	return &NotificationSendHistory{
		NotificationID: notificationID,
		timeStart:      time.Now(),
	}
}

func (t *NotificationSendHistory) End() *NotificationSendHistory {
	t.DurationMs = time.Since(t.timeStart).Milliseconds()
	return t
}
