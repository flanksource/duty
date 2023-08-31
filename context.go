package duty

import (
	"context"

	"gorm.io/gorm"
)

type dbContext interface {
	context.Context
	DB() *gorm.DB
}
