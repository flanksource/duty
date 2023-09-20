package duty

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

type DBContext interface {
	context.Context
	DB() *gorm.DB
	Pool() *pgxpool.Pool
}
