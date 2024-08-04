package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// Artifact represents the artifacts table
type Artifact struct {
	ID                  uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	CheckID             *uuid.UUID `json:"check_id,omitempty"`
	CheckTime           *time.Time `json:"check_time,omitempty" time_format:"postgres_timestamp"`
	PlaybookRunActionID *uuid.UUID `json:"playbook_run_action_id,omitempty"`
	ConnectionID        uuid.UUID  `json:"connection_id,omitempty"`
	Path                string     `json:"path"`
	IsPushed            bool       `json:"is_pushed"`
	IsDataPushed        bool       `json:"is_data_pushed"`
	Filename            string     `json:"filename"`
	Size                int64      `json:"size"` // Size in bytes
	ContentType         string     `json:"content_type,omitempty"`
	Checksum            string     `json:"checksum"`
	CreatedAt           time.Time  `json:"created_at" yaml:"created_at" time_format:"postgres_timestamp"`
	UpdatedAt           time.Time  `json:"updated_at" yaml:"updated_at" time_format:"postgres_timestamp"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty" time_format:"postgres_timestamp"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty" time_format:"postgres_timestamp"`
}

func (a Artifact) TableName() string {
	return "artifacts"
}

func (a Artifact) PK() string {
	return a.ID.String()
}

func (t Artifact) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Artifact
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Artifact, _ int) DBTable { return i }), err
}
