package models

import (
	/*
		"database/sql/driver"

		"github.com/flanksource/duty/types"
	*/
	"github.com/google/uuid"
)

type Person struct {
	ID         uuid.UUID        `json:"id" gorm:"default:generate_ulid()"`
	Name       string           `json:"name,omitempty"`
	Email      string           `json:"email,omitempty"`
	Avatar     string           `json:"avatar,omitempty"`
	Properties PersonProperties `json:"properties,omitempty"`
}

func (person Person) TableName() string {
	return "people"
}

type PersonProperties struct {
	Role string `json:"role,omitempty"`
}

/*
func (p PersonProperties) Value() (driver.Value, error) {
	return types.GenericStructValue(p, true)
}

func (p *PersonProperties) Scan(val any) error {
	return types.GenericStructScan(&p, val)
}*/
