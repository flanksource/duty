package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	k8sDuration "k8s.io/apimachinery/pkg/util/duration"
)

type Latency struct {
	// Percentile99 float64 `json:"p99,omitempty" db:"p99"`
	// Percentile95 float64 `json:"p95,omitempty" db:"p95"`
	// Percentile50 float64 `json:"p50,omitempty" db:"p50"`
	// Avg          float64 `json:"avg,omitempty" db:"mean"`
	// Rolling1H    float64 `json:"rolling1h"`

	Percentile99 float64 `json:"p99,omitempty" db:"p99"`
	Percentile97 float64 `json:"p97,omitempty" db:"p97"`
	Percentile95 float64 `json:"p95,omitempty" db:"p95"`
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

func (l Latency) String() string {
	s := ""
	if l.Percentile99 != 0 {
		s += fmt.Sprintf("p99=%s", age(time.Duration(l.Percentile99)*time.Millisecond))
	}
	if l.Percentile95 != 0 {
		s += fmt.Sprintf("p95=%s", age(time.Duration(l.Percentile95)*time.Millisecond))
	}
	if l.Percentile97 != 0 {
		s += fmt.Sprintf("p97=%s", age(time.Duration(l.Percentile97)*time.Millisecond))
	}
	if l.Rolling1H != 0 {
		s += fmt.Sprintf("rolling1h=%s", age(time.Duration(l.Rolling1H)*time.Millisecond))
	}
	return s
}

func age(d time.Duration) string {
	if d.Milliseconds() == 0 {
		return "0ms"
	}
	if d.Milliseconds() < 1000 {
		return fmt.Sprintf("%0.dms", d.Milliseconds())
	}

	return k8sDuration.HumanDuration(d)
}
