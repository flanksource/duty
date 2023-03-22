package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type LoggingBackend struct {
	ID        uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	Name      string              `json:"name"`
	Labels    types.JSONStringMap `json:"labels" gorm:"type:jsonstringmap"`
	Spec      string              `json:"spec,omitempty"`
	Source    string              `json:"source,omitempty"`
	CreatedAt time.Time           `json:"created_at,omitempty"`
	UpdatedAt time.Time           `json:"updated_at,omitempty"`
	DeletedAt *time.Time          `json:"deleted_at,omitempty"`
}
