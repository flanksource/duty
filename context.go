package duty

import (
	"context"

	"gorm.io/gorm"
)

type DBContext interface {
	context.Context
	DB() *gorm.DB
}
