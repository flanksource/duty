package types

import (
	"context"
	"database/sql/driver"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// +kubebuilder:object:generate=true
type LogSelector struct {
	Name   string            `json:"name,omitempty" yaml:"name,omitempty"`
	Type   string            `json:"type,omitempty" yaml:"type,omitempty" template:"true"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty" template:"true"`
}

type LogSelectors []LogSelector

func (t LogSelectors) Value() (driver.Value, error) {
	return GenericStructValue(t, true)
}

func (t *LogSelectors) Scan(val any) error {
	return GenericStructScan(&t, val)
}

func (t LogSelectors) GormDataType() string {
	return "logSelectors"
}

func (t LogSelectors) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (t LogSelectors) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(t)
}
