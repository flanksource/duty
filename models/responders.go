package models

import (
	"time"

	"github.com/google/uuid"
)

type Responder struct {
	ID           uuid.UUID  `gorm:"primaryKey" json:"id"`
	IncidentID   uuid.UUID  `json:"incident_id"`
	Type         string     `json:"type"`
	Index        *int16     `json:"index,omitempty"`
	PersonID     *uuid.UUID `json:"person_id,omitempty"`
	TeamID       *uuid.UUID `json:"team_id,omitempty"`
	ExternalID   *string    `json:"external_id,omitempty"`
	Properties   *string    `gorm:"type:jsonb;default:null" json:"properties,omitempty"`
	Acknowledged *time.Time `json:"acknowledged,omitempty" time_format:"postgres_timestamp"`
	Resolved     *time.Time `json:"resolved,omitempty" time_format:"postgres_timestamp"`
	Closed       *time.Time `json:"closed,omitempty" time_format:"postgres_timestamp"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
	UpdatedAt    time.Time  `json:"updated_at" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
}

func (r Responder) TableName() string {
	return "responders"
}

func (r Responder) PK() string {
	return r.ID.String()
}

func (r Responder) AsMap(removeFields ...string) map[string]any {
	return asMap(r, removeFields...)
}
