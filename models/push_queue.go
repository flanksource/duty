package models

import (
	"time"

	"github.com/google/uuid"
)

// PushQueue represents the push_queue database table.
type PushQueue struct {
	ID        uuid.UUID `gorm:"column:id"`
	ItemID    string    `gorm:"column:item_id"` // ItemID can contain a single primary key or a composite key separated by a colon ':'
	Table     string    `gorm:"column:table_name"`
	Operation string    `gorm:"column:operation"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (PushQueue) TableName() string {
	return "push_queue"
}
