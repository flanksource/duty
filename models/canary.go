package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Canary struct {
	ID        uuid.UUID `gorm:"default:generate_ulid()"`
	Spec      types.JSON
	Labels    types.JSONStringMap
	Source    string
	Name      string
	Namespace string
	Checks    types.JSONStringMap `gorm:"-"`
	Schedule  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (c Canary) GetCheckID(checkName string) string {
	return c.Checks[checkName]
}

func (c Canary) TableName() string {
	return "canaries"
}
