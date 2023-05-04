package types

import (
	"context"
	"database/sql/driver"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// +kubebuilder:object:generate=true
// ConfigQuery is used to look up and associate
// config items with a component.
type ConfigQuery struct {
	ID         []string          `json:"id,omitempty"`
	Type       string            `json:"type,omitempty"`
	Class      string            `json:"class,omitempty"`
	ExternalID string            `json:"external_id,omitempty"`
	Name       string            `json:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

func (c ConfigQuery) String() string {
	return fmt.Sprintf("id=%v, type=%s, class=%s, external_id=%s, name=%s, namespace=%s, tags=%v",
		c.ID,
		c.Type,
		c.Class,
		c.ExternalID,
		c.Name,
		c.Namespace,
		c.Tags,
	)
}

type ConfigQueries []*ConfigQuery

func (t ConfigQueries) Value() (driver.Value, error) {
	return GenericStructValue(t, true)
}

func (t *ConfigQueries) Scan(val any) error {
	return GenericStructScan(&t, val)
}

func (t ConfigQueries) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (t ConfigQueries) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(t)
}
