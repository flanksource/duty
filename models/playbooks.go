package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

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
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

type PlaybookRun struct {
	ID          uuid.UUID           `gorm:"default:generate_ulid()"`
	PlaybookID  uuid.UUID           `json:"playbook_id"`
	CreatedAt   time.Time           `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW()"`
	StartTime   time.Time           `json:"start_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW()"`
	EndTime     *time.Time          `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	Duration    time.Duration       `json:"duration" gorm:"default:null"`
	Result      types.JSON          `json:"result,omitempty"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty"`
	ComponentID *uuid.UUID          `json:"component_id,omitempty"`
	ConfigID    *uuid.UUID          `json:"config_id,omitempty"`
	Parameters  types.JSONStringMap `json:"parameters,omitempty" gorm:"default:null"`
	Status      PlaybookRunStatus   `json:"status,omitempty"`
	AgentID     *uuid.UUID          `json:"agent_id,omitempty"`
}
