package types

import (
	"context"
	"database/sql/driver"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type Latency struct {
	Percentile99 float64 `json:"p99,omitempty" db:"p99"`
	Percentile95 float64 `json:"p95,omitempty" db:"p95"`
	Percentile50 float64 `json:"p50,omitempty" db:"p50"`
	Avg          float64 `json:"avg,omitempty" db:"mean"`
	Rolling1H    float64 `json:"rolling1h"`
}

func (l Latency) Value() (driver.Value, error) {
	return GenericStructValue(l, true)
}

func (l *Latency) Scan(val any) error {
	return GenericStructScan(&l, val)
}

func (l Latency) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (l Latency) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(l)
}
