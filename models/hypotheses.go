package models

import (
	"time"

	"github.com/google/uuid"
)

type Hypothesis struct {
	ID         uuid.UUID  `json:"id,omitempty" gorm:"default:generate_ulid()"`
	IncidentID uuid.UUID  `json:"incident_id,omitempty"`
	Type       string     `json:"type,omitempty"`
	Title      string     `json:"title,omitempty"`
	Status     string     `json:"status,omitempty"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	TeamID     *uuid.UUID `json:"team_id,omitempty"`
	Owner      *uuid.UUID `json:"owner,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
	CreatedBy  uuid.UUID  `json:"created_by,omitempty"`
}

func (h Hypothesis) PK() string {
	return h.ID.String()
}

func (Hypothesis) TableName() string {
	return "hypotheses"
}

func (h Hypothesis) AsMap(removeFields ...string) map[string]any {
	return asMap(h, removeFields...)
}
