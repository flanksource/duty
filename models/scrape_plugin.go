package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type ScrapePlugin struct {
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace"`
	Spec      types.JSON `json:"spec,omitempty"`
	Source    string     `json:"source,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:create"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (s ScrapePlugin) GetNamespace() string {
	return s.Namespace
}
