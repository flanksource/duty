package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Playbook struct {
	ID          uuid.UUID  `gorm:"default:generate_ulid()"`
	Description string     `json:"description"`
	Spec        types.JSON `json:"spec"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

type PlaybookRun struct {
	ID          uuid.UUID  `gorm:"default:generate_ulid()"`
	PlaybookID  uuid.UUID  `json:"playbook_id"`
	CreatedAt   time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	StartedAt   *time.Time `json:"started_at,omitempty" time_format:"postgres_timestamp"`
	CompletedAt *time.Time `json:"completed_at,omitempty" time_format:"postgres_timestamp"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
}
