package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/flanksource/duty/types"
)

// AccessScope represents a visibility boundary for Guest users
type AccessScope struct {
	ID          uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace,omitempty" gorm:"default:NULL"`
	Description string     `json:"description,omitempty"`
	Source      string     `json:"source"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at" gorm:"<-:false"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"<-:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`

	// Subject fields - exactly one should be set
	PersonID *uuid.UUID `json:"person_id,omitempty"`
	TeamID   *uuid.UUID `json:"team_id,omitempty"`

	// Set of resources, these scopes apply to.
	// Resources can be: configs, playbooks, components, canaries
	Resources pq.StringArray `json:"resources" gorm:"type:text[]"`

	// Array of AccessScopeScope stored as JSONB
	Scopes types.JSON `json:"scopes" gorm:"type:jsonb"`
}

func (AccessScope) TableName() string {
	return "access_scopes"
}
