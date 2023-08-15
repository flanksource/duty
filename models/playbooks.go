package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Playbook struct {
	ID          uuid.UUID `gorm:"default:generate_ulid()"`
	Description string    `json:"description"`
	Spec        types.JSON
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}
