package models

import (
	"database/sql/driver"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (p Person) GetName() string {
	if p.Email != "" {
		return p.Email
	}
	if p.ExternalID != "" {
		return p.ExternalID
	}
	if p.Name != "" {
		return p.Name
	}
	if p.ID != uuid.Nil {
		return p.ID.String()
	}
	return ""
}

func (p Person) PK() string {
	return p.ID.String()
}

func (person Person) TableName() string {
	return "people"
}

func (p *Person) Save(db *gorm.DB) error {
	if p.ID != uuid.Nil {
		return db.Model(p).Clauses(
			clause.Returning{},
		).Save(p).Error
	}
	return db.Model(p).Clauses(
		clause.Returning{},
		clause.OnConflict{
			Columns:     []clause.Column{{Name: "email"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
			UpdateAll:   true,
		}).Create(p).Error
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
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name      string     `json:"name" gorm:"not null"`
	Value     string     `json:"-" gorm:"not null"`
	AutoRenew bool       `json:"auto_renew"`
	PersonID  uuid.UUID  `json:"person_id" gorm:"not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:create"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (AccessToken) TableName() string {
	return "access_tokens"
}

func (a AccessToken) PK() string {
	return a.ID.String()
}
