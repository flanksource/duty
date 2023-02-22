package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// SystemTemplate represents the templates database table
type SystemTemplate struct {
	ID        uuid.UUID `gorm:"default:generate_ulid()"`
	Name      string
	Namespace string
	Labels    types.JSONStringMap
	Spec      types.JSON
	Schedule  *string
	CreatedAt *time.Time `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" time_format:"postgres_timestamp"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (cr SystemTemplate) TableName() string {
	return "templates"
}
