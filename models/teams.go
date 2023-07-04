package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Team struct {
	ID        uuid.UUID  `gorm:"default:generate_ulid(), primaryKey"`
	Name      string     `gorm:"not null" json:"name"`
	Icon      string     `json:"icon,omitempty"`
	Spec      types.JSON `json:"spec,omitempty"`
	Source    string     `json:"source,omitempty"`
	CreatedBy uuid.UUID  `gorm:"not null" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type TeamComponent struct {
	TeamID      uuid.UUID `json:"team_id" gorm:"primaryKey"`
	ComponentID uuid.UUID `json:"component_id" gorm:"primaryKey"`
	Role        *string   `json:"role,omitempty"`
	SelectorID  *string   `json:"selector_id,omitempty"`
}
