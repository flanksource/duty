package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// Plugin is the persisted form of a Mission Control Plugin CRD. It is the
// source of truth for "which plugins are configured"; the in-memory registry
// is rehydrated from this table on startup. The full v1.PluginSpec is held
// verbatim in the spec jsonb column.
type Plugin struct {
	ID            uuid.UUID  `gorm:"primaryKey;column:id;default:generate_ulid()" json:"id"`
	Name          string     `gorm:"column:name;not null" json:"name"`
	Namespace     string     `gorm:"column:namespace;not null;default:default" json:"namespace"`
	Source        string     `gorm:"column:source" json:"source,omitempty"`
	Spec          types.JSON `gorm:"column:spec;not null" json:"spec"`
	InstalledPath string     `gorm:"column:installed_path" json:"installed_path,omitempty"`
	PluginVersion string     `gorm:"column:plugin_version" json:"plugin_version,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;default:now();<-:create" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;default:now()" json:"updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (p Plugin) TableName() string {
	return "plugins"
}
