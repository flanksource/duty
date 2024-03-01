package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// Topology represents the topologies database table
type Topology struct {
	ID        uuid.UUID           `gorm:"default:generate_ulid()"`
	AgentID   uuid.UUID           `json:"agent_id"`
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	Labels    types.JSONStringMap `json:"labels,omitempty"  gorm:"default:null"`
	Source    string              `json:"source"`
	Spec      types.JSON          `gorm:"default:null"`
	Schedule  *string             `json:"schedule,omitempty"`
	CreatedBy *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt *time.Time          `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	UpdatedAt *time.Time          `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"autoUpdateTime:false"`
	DeletedAt *time.Time          `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (t Topology) PK() string {
	return t.ID.String()
}

func (Topology) TableName() string {
	return "topologies"
}

func (t *Topology) AsMap(removeFields ...string) map[string]any {
	return asMap(t, removeFields...)
}
