package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID                uuid.UUID  `json:"id" gorm:"primaryKey"`
	CreatedBy         uuid.UUID  `json:"created_by,omitempty"`
	Comment           string     `json:"comment,omitempty"`
	ExternalID        *string    `json:"external_id,omitempty"`
	ExternalCreatedBy *string    `json:"external_created_by,omitempty"`
	IncidentID        uuid.UUID  `json:"incident_id,omitempty"`
	ResponderID       *uuid.UUID `json:"responder_id,omitempty"`
	HypothesisID      *uuid.UUID `json:"hypothesis_id,omitempty"`
	Read              []int16    `json:"read,omitempty" gorm:"type:smallint[]"`
	CreatedAt         time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
	UpdatedAt         time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
}

func (c Comment) TableName() string {
	return "comments"
}

func (c Comment) PK() string {
	return c.ID.String()
}

func (c Comment) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}
