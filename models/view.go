package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/types"
)

// View represents the views database table
type View struct {
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace" gorm:"default:NULL"`
	Spec      types.JSON `json:"spec"`
	Source    string     `json:"source" gorm:"default:KubernetesCRD"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:create"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
	LastRan   *time.Time `json:"last_ran,omitempty" gorm:"default:NULL"`
	Error     *string    `json:"error,omitempty" gorm:"default:NULL"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (v View) PK() string {
	return v.ID.String()
}

func (View) TableName() string {
	return "views"
}

func (v View) AsMap(removeFields ...string) map[string]any {
	return asMap(v, removeFields...)
}

func (v View) GetNamespace() string {
	return v.Namespace
}

type PanelResult struct {
	ViewID   uuid.UUID `json:"view_id" gorm:"primaryKey"`
	AgentID  uuid.UUID `json:"agent_id"`
	IsPushed bool      `json:"is_pushed" gorm:"default:false"`

	// Results is a JSON array of panel results
	Results types.JSON `json:"results"`
}

func (PanelResult) TableName() string {
	return "panel_results"
}

func (v PanelResult) PK() string {
	return v.ViewID.String()
}
