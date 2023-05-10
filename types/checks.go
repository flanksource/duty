package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type TestThreshold struct {
	High     string `yaml:"high,omitempty" json:"high,omitempty"`
	Low      string `yaml:"low,omitempty" json:"low,omitempty"`
	Critical string `yaml:"critical,omitempty" json:"critical,omitempty"`
}

func (t TestThreshold) Value() (driver.Value, error) {
	return GenericStructValue(t, true)
}

func (t *TestThreshold) Scan(val any) error {
	return GenericStructScan(&t, val)
}

func (t TestThreshold) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case SqliteType:
		return JSONType
	case PostgresType:
		return JSONBType
	case SQLServerType:
		return NVarcharType
	}
	return ""
}

func (t TestThreshold) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(t)
	return gorm.Expr("?", string(data))
}
