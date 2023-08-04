package models

import (
	"database/sql/driver"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Person struct {
	ID         uuid.UUID        `json:"id" gorm:"default:generate_ulid()"`
	Name       string           `json:"name,omitempty"`
	Email      string           `json:"email,omitempty" gorm:"default:null"`
	Type       string           `json:"type,omitempty" gorm:"default:null"`
	Avatar     string           `json:"avatar,omitempty" gorm:"default:null"`
	Properties PersonProperties `json:"properties,omitempty" gorm:"default:null"`
}

func (person Person) TableName() string {
	return "people"
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
	ID        uuid.UUID `json:"id" gorm:"default:generate_ulid()"`
	Value     string
	PersonID  uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (AccessToken) TableName() string {
	return "access_tokens"
}
