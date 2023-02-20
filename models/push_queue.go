package models

import (
	"time"

	"github.com/google/uuid"
)

// PushQueue represents the push_queue database table.
type PushQueue struct {
	ID        uuid.UUID `gorm:"column:id"`
	ItemID    uuid.UUID `gorm:"column:item_id"`
	Table     string    `gorm:"column:table_name"`
	Operation string    `gorm:"column:operation"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (PushQueue) TableName() string {
	return "push_queue"
}
