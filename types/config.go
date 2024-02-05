package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

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

func (c ConfigQuery) ToResourceSelector() ResourceSelector {
	var labelSelectors []string
	for k, v := range c.Tags {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", k, v))
	}

	var fieldSelectors []string
	if c.ExternalID != "" {
		fieldSelectors = append(fieldSelectors, fmt.Sprintf("external_id=%s", c.ExternalID))
	}
	if c.Class != "" {
		fieldSelectors = append(fieldSelectors, fmt.Sprintf("config_class=%s", c.Class))
	}

	rs := ResourceSelector{
		Name:          c.Name,
		Namespace:     c.Namespace,
		FieldSelector: strings.Join(fieldSelectors, ","),
		LabelSelector: strings.Join(labelSelectors, ","),
	}
	if len(c.ID) > 0 {
		rs.ID = c.ID[0] // TODO: Add support for multiple ids in the resource selector (or remove multiple IDs from the config query)
	}
	if len(c.Type) > 0 {
		rs.Types = Items{c.Type}
	}

	return rs
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

func (t ConfigQueries) GormDataType() string {
	return "configQueries"
}

func (t ConfigQueries) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (t ConfigQueries) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(t)
}

// ToJSONMap converts the struct to map[string]interface{}
func (c ConfigQuery) ToJSONMap() map[string]interface{} {
	m := make(map[string]interface{})
	b, _ := json.Marshal(&c)
	_ = json.Unmarshal(b, &m)
	return m
}
