package models

import (
	"database/sql/driver"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Person struct {
	ID         uuid.UUID        `json:"id" gorm:"default:generate_ulid()"`
	Name       string           `json:"name"`
	Email      string           `json:"email,omitempty" gorm:"default:null"`
	Type       string           `json:"type,omitempty" gorm:"default:null"`
	Avatar     string           `json:"avatar,omitempty" gorm:"default:null"`
	ExternalID string           `json:"external_id,omitempty" gorm:"default:null"`
	Properties PersonProperties `json:"properties,omitempty" gorm:"default:null"`
}

func (person Person) TableName() string {
	return "people"
}

func (person Person) AsMap(removeFields ...string) map[string]any {
	return asMap(person, removeFields...)

}

type PersonProperties struct {
	Role string `json:"role,omitempty"`
}

func (p PersonProperties) Value() (driver.Value, error) {
	return types.GenericStructValue(p, true)
}

func (p *PersonProperties) Scan(val any) error {
	return types.GenericStructScan(&p, val)
}

type AccessToken struct {
	ID        uuid.UUID `gorm:"default:generate_ulid()"`
	Name      string    `gorm:"not null"`
	Value     string    `gorm:"not null"`
	PersonID  uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (AccessToken) TableName() string {
	return "access_tokens"
}
