package models

import (
	"time"

	"github.com/google/uuid"
)

type ConfigBackup struct {
	ID           uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	ConfigItemID uuid.UUID  `json:"config_item_id"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"`
	Error        *string    `json:"error,omitempty"`
	Size         int64      `json:"size"`
}

type ConfigBackupRestore struct {
	ID             uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	ConfigBackupID uuid.UUID  `json:"config_backup_id"`
	ConfigItemID   uuid.UUID  `json:"config_item_id"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Status         string     `json:"status"`
	Error          *string    `json:"error,omitempty"`
}
