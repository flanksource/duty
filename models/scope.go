package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/flanksource/duty/types"
)

// Scope represents a collection of resources of a single type
// that can be used for access control and permissions
type Scope struct {
	ID          uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace,omitempty" gorm:"default:NULL"`
	Description string     `json:"description,omitempty"`
	Source      string     `json:"source"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at" gorm:"<-:false"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"<-:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`

	// Targets is an array of scope targets stored as JSONB
	// Each target contains exactly one resource type key (config, component, playbook, canary, or *)
	// with a selector containing: agent, name, tagSelector fields
	Targets types.JSON `json:"targets" gorm:"type:jsonb"`
}

func (Scope) TableName() string {
	return "scopes"
}

func (s Scope) PK() string {
	return s.ID.String()
}
