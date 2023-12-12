package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// Topology represents the topologies database table
type Topology struct {
	ID        uuid.UUID `gorm:"default:generate_ulid()"`
	AgentID   uuid.UUID `json:"agent_id,omitempty"`
	Name      string
	Namespace string
	Labels    types.JSONStringMap `json:"labels,omitempty"  gorm:"default:null"`
	Source    string              `json:"source"`
	Spec      types.JSON          `gorm:"default:null"`
	Schedule  *string
	CreatedAt *time.Time `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" time_format:"postgres_timestamp"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (Topology) TableName() string {
	return "topologies"
}

func (t *Topology) AsMap(removeFields ...string) map[string]any {
	return asMap(t, removeFields...)
}
