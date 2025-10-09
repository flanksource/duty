package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ScopeBinding binds human subjects (persons or teams) to a set of Scopes
type ScopeBinding struct {
	ID          uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace,omitempty" gorm:"default:NULL"`
	Description string     `json:"description,omitempty"`
	Source      string     `json:"source"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at" gorm:"<-:false"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"<-:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`

	// Persons is an array of person emails
	Persons pq.StringArray `json:"persons" gorm:"type:text[]"`

	// Teams is an array of team names
	Teams pq.StringArray `json:"teams" gorm:"type:text[]"`

	// Scopes is an array of scope names (must be in same namespace)
	Scopes pq.StringArray `json:"scopes" gorm:"type:text[]"`
}

func (ScopeBinding) TableName() string {
	return "scope_bindings"
}
