package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type Uptime struct {
	Passed   int        `json:"passed"`
	Failed   int        `json:"failed"`
	P100     float64    `json:"p100,omitempty"`
	LastPass *time.Time `json:"last_pass,omitempty"`
	LastFail *time.Time `json:"last_fail,omitempty"`
}

func (u Uptime) String() string {
	if u.Passed == 0 && u.Failed == 0 {
		return ""
	}
	if u.Passed == 0 {
		return fmt.Sprintf("0/%d 0%%", u.Failed)
	}
	percentage := 100.0 * (1 - (float64(u.Failed) / float64(u.Passed+u.Failed)))
	return fmt.Sprintf("%d/%d (%0.1f%%)", u.Passed, u.Passed+u.Failed, percentage)
}

func (u Uptime) Value() (driver.Value, error) {
	return GenericStructValue(u, true)
}

func (u *Uptime) Scan(val any) error {
	return GenericStructScan(&u, val)
}

func (u Uptime) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (u Uptime) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(u)
}
