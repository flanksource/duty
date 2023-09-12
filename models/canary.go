package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Canary struct {
	ID        uuid.UUID           `json:"id" yaml:"id" gorm:"default:generate_ulid()"`
	Name      string              `json:"name" yaml:"name"`
	Namespace string              `json:"namespace" yaml:"namespace"`
	AgentID   uuid.UUID           `json:"agent_id" yaml:"agent_id"`
	Spec      types.JSON          `json:"spec" yaml:"spec"`
	Labels    types.JSONStringMap `json:"labels,omitempty" yaml:"labels,omitempty"`
	Source    string              `json:"source,omitempty" yaml:"source,omitempty"`
	Checks    types.JSONStringMap `gorm:"-" json:"checks,omitempty" yaml:"checks,omitempty"`
	CreatedAt time.Time           `json:"created_at" yaml:"created_at" time_format:"postgres_timestamp"`
	UpdatedAt time.Time           `json:"updated_at" yaml:"updated_at" time_format:"postgres_timestamp"`
	DeletedAt *time.Time          `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (c Canary) GetCheckID(checkName string) string {
	return c.Checks[checkName]
}

func (c Canary) TableName() string {
	return "canaries"
}

func (c Canary) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}
