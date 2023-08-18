package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// PlaybookRunStatus are statuses for a playbook run and its actions.
type PlaybookRunStatus string

const (
	PlaybookRunStatusScheduled PlaybookRunStatus = "scheduled"
	PlaybookRunStatusRunning   PlaybookRunStatus = "running"
	PlaybookRunStatusCancelled PlaybookRunStatus = "cancelled"
	PlaybookRunStatusFailed    PlaybookRunStatus = "failed"
	PlaybookRunStatusCompleted PlaybookRunStatus = "completed"
)

type Playbook struct {
	ID        uuid.UUID  `gorm:"default:generate_ulid()"`
	Name      string     `json:"name"`
	Spec      types.JSON `json:"spec"`
	Source    string     `json:"source"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

type PlaybookRun struct {
	ID          uuid.UUID           `gorm:"default:generate_ulid()"`
	PlaybookID  uuid.UUID           `json:"playbook_id"`
	Status      PlaybookRunStatus   `json:"status,omitempty"`
	CreatedAt   time.Time           `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	StartTime   time.Time           `json:"start_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW(), not null"`
	EndTime     *time.Time          `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty"`
	ComponentID *uuid.UUID          `json:"component_id,omitempty"`
	ConfigID    *uuid.UUID          `json:"config_id,omitempty"`
	Parameters  types.JSONStringMap `json:"parameters,omitempty" gorm:"default:null"`
	AgentID     *uuid.UUID          `json:"agent_id,omitempty"`
}

type PlaybookRunAction struct {
	ID            uuid.UUID         `gorm:"default:generate_ulid()"`
	Name          string            `json:"name" gorm:"not null"`
	PlaybookRunID uuid.UUID         `json:"playbook_run_id"`
	Status        PlaybookRunStatus `json:"status,omitempty"`
	StartTime     time.Time         `json:"start_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW(), not null"`
	EndTime       *time.Time        `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	Result        types.JSON        `json:"result,omitempty" gorm:"default:null"`
	Error         string            `json:"error,omitempty" gorm:"default:null"`
}
