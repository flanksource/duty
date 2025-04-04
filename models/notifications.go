package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// Notification represents the notifications table
type Notification struct {
	ID             uuid.UUID           `json:"id"`
	Name           string              `json:"name"`
	Namespace      string              `json:"namespace,omitempty"`
	Events         pq.StringArray      `json:"events" gorm:"type:[]text"`
	Title          string              `json:"title,omitempty"`
	Template       string              `json:"template,omitempty"`
	Filter         string              `json:"filter,omitempty"`
	Properties     types.JSONStringMap `json:"properties,omitempty"`
	Source         string              `json:"source"`
	RepeatInterval string              `json:"repeat_interval,omitempty"`
	GroupBy        pq.StringArray      `json:"group_by" gorm:"type:[]text"`
	CreatedBy      *uuid.UUID          `json:"created_by,omitempty"`
	UpdatedAt      time.Time           `json:"updated_at" time_format:"postgres_timestamp" gorm:"<-:false"`
	CreatedAt      time.Time           `json:"created_at" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt      *time.Time          `json:"deleted_at,omitempty"`

	// List of inhibition config
	Inhibitions types.JSON `json:"inhibitions,omitempty" gorm:"default:NULL"`

	// Receipients
	PlaybookID     *uuid.UUID `json:"playbook_id,omitempty"`
	PersonID       *uuid.UUID `json:"person_id,omitempty"`
	TeamID         *uuid.UUID `json:"team_id,omitempty"`
	CustomServices types.JSON `json:"custom_services,omitempty" gorm:"column:custom_services"`

	// Fallback Receipients
	FallbackPlaybookID     *uuid.UUID     `json:"fallback_playbook_id,omitempty"`
	FallbackPersonID       *uuid.UUID     `json:"fallback_person_id,omitempty"`
	FallbackTeamID         *uuid.UUID     `json:"fallback_team_id,omitempty"`
	FallbackCustomServices types.JSON     `json:"fallback_custom_services,omitempty"`
	FallbackDelay          *time.Duration `json:"fallback_delay,omitempty"`

	// Duration to wait before re-evaluating health of the resource.
	WaitFor *time.Duration `json:"wait_for,omitempty"`

	// Duration to wait after triggering incremental scrape for kubernetes config.
	// Works together with waitFor duration.
	WaitForEvalPeriod *time.Duration `json:"wait_for_eval_period,omitempty"`

	// Error stores errors in notification filters (if any).
	Error *string `json:"error,omitempty"`
}

func (n Notification) HasFallbackSet() bool {
	return n.FallbackTeamID != nil || n.FallbackPersonID != nil || len(n.FallbackCustomServices) != 0 || n.FallbackPlaybookID != nil
}

func (n Notification) TableName() string {
	return "notifications"
}

func (n Notification) PK() string {
	return n.ID.String()
}

func (n *Notification) HasRecipients() bool {
	return n.TeamID != nil || n.PersonID != nil || len(n.CustomServices) != 0 || n.PlaybookID != nil
}

func (n Notification) AsMap(removeFields ...string) map[string]any {
	return asMap(n, removeFields...)
}

func (n Notification) GetNamespace() string {
	return n.Namespace
}

const (
	NotificationStatusError          = "error"
	NotificationStatusSent           = "sent"
	NotificationStatusPending        = "pending" // delayed due to waitFor evaluation
	NotificationStatusSkipped        = "skipped" // due to waitFor evaluation
	NotificationStatusSilenced       = "silenced"
	NotificationStatusRepeatInterval = "repeat-interval"

	// notification is inhibited by another notification
	NotificationStatusInhibited = "inhibited"

	// an event was triggered and the notification is waiting for the playbook run to be triggered.
	NotificationStatusPendingPlaybookRun = "pending_playbook_run"

	// A playbook is currently in progress
	NotificationStatusPendingPlaybookCompletion = "pending_playbook_completion"

	// health related notifications of kubernetes config items get into this state
	// to wait for the incremental scraper to re-evaluate the health.
	NotificationStatusEvaluatingWaitFor = "evaluating-waitfor"

	// Attempting delivery through a fallback channel
	NotificationStatusAttemptingFallback = "attempting_fallback"

	// Grouped with other notifications
	NotificationStatusGrouped = "grouped"
)

type NotificationSendHistory struct {
	ID             uuid.UUID `json:"id,omitempty" gorm:"default:generate_ulid()"`
	NotificationID uuid.UUID `json:"notification_id"`
	Body           *string   `json:"body,omitempty"`
	Error          *string   `json:"error,omitempty"`
	DurationMillis int64     `json:"duration_millis,omitempty"`
	CreatedAt      time.Time `json:"created_at" time_format:"postgres_timestamp"`
	Status         string    `json:"status,omitempty"`

	// payload holds in original event properties for delayed/pending notifications
	Payload types.JSONStringMap `json:"payload,omitempty"`

	NotBefore *time.Time `json:"notBefore,omitempty"`

	// number of retries of pending notifications
	Retries int `json:"retries,omitempty" gorm:"default:null"`

	// Notifications that were silenced or blocked by repeat intervals
	// use this counter.
	Count int `json:"count" gorm:"default:1"`

	// Notifications that were silenced or blocked by repeat intervals
	// use this as the first observed timestamp.
	FirstObserved time.Time `json:"first_observed" gorm:"<-:false"`

	// Name of the original event that caused this notification
	SourceEvent string `json:"source_event"`

	// ID of the resource this notification is for
	ResourceID uuid.UUID `json:"resource_id"`

	// ID of the team this notification was dispatched to.
	TeamID *uuid.UUID `json:"team_id,omitempty"`

	// ID of the person this notification was dispatched to.
	PersonID *uuid.UUID `json:"person_id,omitempty"`

	// ID of the connection this notification was dispatched to.
	ConnectionID *uuid.UUID `json:"connection_id,omitempty"`

	// The run created by this notification
	PlaybookRunID *uuid.UUID `json:"playbook_run_id,omitempty"`

	// The notification silence that silenced this notification.
	SilencedBy *uuid.UUID `json:"silenced_by,omitempty"`

	// Hash for grouping resources with same message
	GroupByHash string `json:"group_by_hash,omitempty"`

	// ID of the original send history this notification history is a fallback of.
	ParentID *uuid.UUID `json:"parent_id,omitempty"`

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

func (t *NotificationSendHistory) WithStartTime(s time.Time) *NotificationSendHistory {
	t.timeStart = s
	return t
}

func (t *NotificationSendHistory) PendingPlaybookRun() *NotificationSendHistory {
	t.Status = NotificationStatusPendingPlaybookRun
	return t.End()
}

func (t *NotificationSendHistory) Sent() *NotificationSendHistory {
	t.Status = NotificationStatusSent
	return t.End()
}

func (t *NotificationSendHistory) End() *NotificationSendHistory {
	t.DurationMillis = time.Since(t.timeStart).Milliseconds()
	return t
}

type NotificationSilenceResource struct {
	ConfigID    *string `json:"config_id,omitempty"`
	CanaryID    *string `json:"canary_id,omitempty"`
	ComponentID *string `json:"component_id,omitempty"`
	CheckID     *string `json:"check_id,omitempty"`
}

func (t NotificationSilenceResource) Empty() bool {
	return lo.FromPtr(t.ConfigID) == "" &&
		lo.FromPtr(t.CanaryID) == "" &&
		lo.FromPtr(t.ComponentID) == "" &&
		lo.FromPtr(t.CheckID) == ""
}

type NotificationSilence struct {
	NotificationSilenceResource `json:",inline" yaml:",inline"`

	ID          uuid.UUID           `json:"id"  gorm:"default:generate_ulid()"`
	Namespace   string              `json:"namespace,omitempty" gorm:"default:NULL"`
	Name        string              `json:"name,omitempty"`
	Filter      types.CelExpression `json:"filter,omitempty" gorm:"default:NULL"`
	From        *time.Time          `json:"from,omitempty"`
	Until       *time.Time          `json:"until,omitempty"`
	Source      string              `json:"source"`
	Recursive   bool                `json:"recursive"`
	Description *string             `json:"description,omitempty" gorm:"default:NULL"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt   time.Time           `json:"created_at" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt   time.Time           `json:"updated_at" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`

	Selectors types.JSON `json:"selectors,omitempty" gorm:"default:NULL"`

	// Error contains cel expression error in the filter
	Error *string `json:"error,omitempty"`
}

func (n NotificationSilence) AsMap(removeFields ...string) map[string]any {
	return asMap(n, removeFields...)
}

func (t *NotificationSilence) TableName() string {
	return "notification_silences"
}

func (t NotificationSilence) GetNamespace() string {
	return t.Namespace
}

// GenerateFallbackAttempt creates a new notification history record
// based on the provided history for retrying with fallback recipients.
func GenerateFallbackAttempt(db *gorm.DB, notification Notification, history NotificationSendHistory) error {
	// We need to create a new payload whose recipient points towards the fallback recipients
	payload := make(types.JSONStringMap)
	for k, v := range history.Payload {
		payload[k] = v
	}

	if notification.FallbackTeamID != nil {
		payload["team_id"] = notification.FallbackTeamID.String()
	}
	if notification.FallbackPersonID != nil {
		payload["person_id"] = notification.FallbackPersonID.String()
	}
	if notification.FallbackPlaybookID != nil {
		payload["playbook_id"] = notification.FallbackPlaybookID.String()
	}
	if len(notification.CustomServices) != 0 {
		payload["custom_service"] = string(notification.FallbackCustomServices)
	}

	newHistory := NotificationSendHistory{
		Payload:        payload,
		NotificationID: history.NotificationID,
		Status:         NotificationStatusAttemptingFallback,
		FirstObserved:  history.FirstObserved,
		SourceEvent:    history.SourceEvent,
		ParentID:       &history.ID,
		ResourceID:     history.ResourceID,
		NotBefore:      lo.ToPtr(time.Now()),
	}

	if notification.FallbackDelay != nil {
		newHistory.NotBefore = lo.ToPtr(time.Now().Add(*notification.FallbackDelay))
	}

	return db.Create(&newHistory).Error
}
