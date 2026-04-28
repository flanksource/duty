package models

import (
	"context"
	stdsql "database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	PropertyCreatorTypeScraper = "scraper"
	PropertyCreatorTypePerson  = "person"
)

type ConfigItemProperty struct {
	types.Property

	CreatedBy   string `json:"created_by,omitempty"`
	CreatorType string `json:"creator_type,omitempty"`
}

type ConfigItemProperties []*ConfigItemProperty

func NewConfigItemProperties(props types.Properties) ConfigItemProperties {
	if props == nil {
		return nil
	}

	out := make(ConfigItemProperties, len(props))
	for i, prop := range props {
		if prop == nil {
			continue
		}
		out[i] = &ConfigItemProperty{Property: *prop.DeepCopy()}
	}
	return out
}

func (p ConfigItemProperties) AsProperties() types.Properties {
	if p == nil {
		return nil
	}

	out := make(types.Properties, len(p))
	for i, prop := range p {
		if prop == nil {
			continue
		}
		out[i] = prop.Property.DeepCopy()
	}
	return out
}

func (m ConfigItemProperties) MarshalJSON() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	t := ([]*ConfigItemProperty)(m)
	return json.Marshal(t)
}

func (m *ConfigItemProperties) UnmarshalJSON(b []byte) error {
	t := []*ConfigItemProperty{}
	err := json.Unmarshal(b, &t)
	*m = ConfigItemProperties(t)
	return err
}

func (p ConfigItemProperties) AsJSON() []byte {
	if len(p) == 0 {
		return []byte("[]")
	}
	data, err := json.Marshal(p)
	if err != nil {
		logger.Errorf("Error marshalling config item properties: %v", err)
	}
	return data
}

func (p ConfigItemProperties) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return p.AsJSON(), nil
}

func (p *ConfigItemProperties) Scan(val interface{}) error {
	if val == nil {
		*p = make(ConfigItemProperties, 0)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal config item properties value:", val))
	}
	return json.Unmarshal(ba, p)
}

func (ConfigItemProperties) GormDataType() string {
	return "config_item_properties"
}

func (ConfigItemProperties) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "TEXT"
	case "postgres":
		return "JSONB"
	case "sqlserver":
		return "NVARCHAR(MAX)"
	}
	return ""
}

func (p ConfigItemProperties) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(p)
	return gorm.Expr("?", data)
}

type UpdateConfigItemPropertiesResult struct {
	Changed    bool
	Properties ConfigItemProperties
}

func UpdateConfigItemPropertiesForCreator(tx *gorm.DB, configID uuid.UUID, creatorType string, createdBy uuid.UUID, incoming types.Properties) (UpdateConfigItemPropertiesResult, error) {
	incomingJSON := incoming.AsJSON()

	var result struct {
		Changed    bool
		Properties string
	}
	if err := tx.Raw(`SELECT changed, properties FROM update_config_item_properties_for_creator(@configID, @creatorType, @createdBy, CAST(@incoming AS jsonb))`,
		stdsql.Named("configID", configID),
		stdsql.Named("creatorType", creatorType),
		stdsql.Named("createdBy", createdBy),
		stdsql.Named("incoming", string(incomingJSON)),
	).Scan(&result).Error; err != nil {
		return UpdateConfigItemPropertiesResult{}, err
	}

	var merged ConfigItemProperties
	if result.Properties != "" {
		if err := json.Unmarshal([]byte(result.Properties), &merged); err != nil {
			return UpdateConfigItemPropertiesResult{}, fmt.Errorf("unmarshal merged properties: %w", err)
		}
	}

	return UpdateConfigItemPropertiesResult{Changed: result.Changed, Properties: merged}, nil
}
