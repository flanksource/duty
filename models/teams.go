package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Team struct {
	ID        uuid.UUID  `gorm:"default:generate_ulid()"`
	Name      string     `gorm:"not null" json:"name"`
	Icon      string     `json:"icon,omitempty"`
	Spec      types.JSON `json:"spec,omitempty"`
	Source    string     `json:"source,omitempty"`
	CreatedBy uuid.UUID  `gorm:"not null" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (t Team) TableName() string {
	return "teams"
}

func (t Team) PK() string {
	return t.ID.String()
}

func (t *Team) AsMap(removeFields ...string) map[string]any {
	return asMap(t, removeFields...)
}
