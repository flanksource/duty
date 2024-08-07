package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// PlaybookRunStatus are statuses for a playbook run and its actions.
type PlaybookRunStatus string

const (
	PlaybookRunStatusCancelled PlaybookRunStatus = "cancelled"
	PlaybookRunStatusCompleted PlaybookRunStatus = "completed"
	PlaybookRunStatusFailed    PlaybookRunStatus = "failed"
	PlaybookRunStatusPending   PlaybookRunStatus = "pending" // pending approval
	PlaybookRunStatusRunning   PlaybookRunStatus = "running"
	PlaybookRunStatusScheduled PlaybookRunStatus = "scheduled"
	PlaybookRunStatusSleeping  PlaybookRunStatus = "sleeping"
	PlaybookRunStatusWaiting   PlaybookRunStatus = "waiting" // waiting for a consumer
)

// PlaybookRunStatus are statuses for a playbook run and its actions.
type PlaybookActionStatus string

const (
	PlaybookActionStatusCompleted PlaybookActionStatus = "completed"
	PlaybookActionStatusFailed    PlaybookActionStatus = "failed"
	PlaybookActionStatusRunning   PlaybookActionStatus = "running"
	PlaybookActionStatusScheduled PlaybookActionStatus = "scheduled"
	PlaybookActionStatusSkipped   PlaybookActionStatus = "skipped"
	PlaybookActionStatusSleeping  PlaybookActionStatus = "sleeping"
)

var PlaybookActionFinalStates = []PlaybookActionStatus{
	PlaybookActionStatusFailed,
	PlaybookActionStatusCompleted,
	PlaybookActionStatusSkipped,
}

func (p Playbook) TableName() string {
	return "playbooks"
}

func (p Playbook) PK() string {
	return p.ID.String()
}

var (
	PlaybookRunStatusExecutingGroup = []PlaybookRunStatus{
		PlaybookRunStatusRunning,
		PlaybookRunStatusScheduled,
		PlaybookRunStatusCompleted,
	}
)

type Playbook struct {
	ID          uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Namespace   string     `json:"namespace"`
	Name        string     `json:"name"`
	Icon        string     `json:"icon,omitempty"`
	Description string     `json:"description,omitempty"`
	Spec        types.JSON `json:"spec"`
	Source      string     `json:"source"`
	Category    string     `json:"category"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (p Playbook) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookRun struct {
	ID            uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	PlaybookID    uuid.UUID           `json:"playbook_id"`
	Status        PlaybookRunStatus   `json:"status,omitempty"`
	CreatedAt     time.Time           `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	StartTime     *time.Time          `json:"start_time,omitempty" time_format:"postgres_timestamp"`
	ScheduledTime time.Time           `json:"scheduled_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW(), NOT NULL"`
	EndTime       *time.Time          `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	CreatedBy     *uuid.UUID          `json:"created_by,omitempty"`
	ComponentID   *uuid.UUID          `json:"component_id,omitempty"`
	CheckID       *uuid.UUID          `json:"check_id,omitempty"`
	ConfigID      *uuid.UUID          `json:"config_id,omitempty"`
	Error         *string             `json:"error,omitempty"`
	Parameters    types.JSONStringMap `json:"parameters,omitempty" gorm:"default:null"`
	Request       types.JSONMap       `json:"request,omitempty" gorm:"default:null"`
	AgentID       *uuid.UUID          `json:"agent_id,omitempty"`
}

func (p PlaybookRun) TableName() string {
	return "playbook_runs"
}

func (p PlaybookRun) PK() string {
	return p.ID.String()
}

func (p PlaybookRun) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookRunAction struct {
	ID            uuid.UUID            `json:"id" gorm:"default:generate_ulid()"`
	Name          string               `json:"name" gorm:"not null"`
	PlaybookRunID uuid.UUID            `json:"playbook_run_id"`
	Status        PlaybookActionStatus `json:"status,omitempty"`
	ScheduledTime time.Time            `json:"scheduled_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW(), NOT NULL"`
	StartTime     time.Time            `json:"start_time,omitempty" time_format:"postgres_timestamp"  gorm:"default:NOW(), NOT NULL"`
	EndTime       *time.Time           `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	Result        types.JSONMap        `json:"result,omitempty" gorm:"default:null"`
	Error         *string              `json:"error,omitempty" gorm:"default:null"`
	IsPushed      bool                 `json:"is_pushed"`
	AgentID       *uuid.UUID           `json:"agent_id,omitempty"`
}

func (p PlaybookRunAction) TableName() string {
	return "playbook_run_actions"
}

func (p PlaybookRunAction) PK() string {
	return p.ID.String()
}

func (p PlaybookRunAction) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookApproval struct {
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	RunID     uuid.UUID  `json:"run_id"`
	PersonID  *uuid.UUID `json:"person_id,omitempty"`
	TeamID    *uuid.UUID `json:"team_id,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:false"`
}

func (p PlaybookApproval) TableName() string {
	return "playbook_approvals"
}

func (p PlaybookApproval) PK() string {
	return p.ID.String()
}

func (p PlaybookApproval) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookActionAgentData struct {
	ActionID   uuid.UUID  `json:"action_id"`
	RunID      uuid.UUID  `json:"run_id"`
	PlaybookID uuid.UUID  `json:"playbook_id"`
	Spec       types.JSON `json:"spec"`
	Env        types.JSON `json:"env,omitempty"`
}

func (t *PlaybookActionAgentData) TableName() string {
	return "playbook_action_agent_data"
}
