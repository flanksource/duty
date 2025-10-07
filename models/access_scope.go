package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/flanksource/duty/types"
)

// AccessScope represents a visibility boundary for human subjects
type AccessScope struct {
	ID          uuid.UUID `json:"id" gorm:"default:generate_ulid()"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace,omitempty"`
	Description string    `json:"description,omitempty"`

	// Subject fields - exactly one should be set
	SubPersonID *uuid.UUID `json:"sub_person_id,omitempty"`
	SubTeamID   *uuid.UUID `json:"sub_team_id,omitempty"`

	Resources []string       `json:"resources" gorm:"type:text[]"` // Array of resource types
	Scopes    types.JSON     `json:"scopes" gorm:"type:jsonb"`     // Array of AccessScopeScope stored as JSONB
	Source    string         `json:"source"`
	CreatedAt time.Time      `json:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (AccessScope) TableName() string {
	return "access_scopes"
}
