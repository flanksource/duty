package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Application struct {
	ID          uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	Name        string              `json:"name"`
	Namespace   string              `json:"namespace,omitempty" gorm:"default:NULL"`
	Description string              `json:"description,omitempty" gorm:"default:NULL"`
	Spec        string              `json:"spec"`
	Source      string              `json:"source,omitempty"`
	Labels      types.JSONStringMap `json:"labels,omitempty" gorm:"default:NULL"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty" gorm:"default:NULL"`
	CreatedAt   time.Time           `json:"created_at" gorm:"<-:create"`
	UpdatedAt   *time.Time          `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`
}

func (a Application) PK() string {
	return a.ID.String()
}

func (a Application) TableName() string {
	return "applications"
}
