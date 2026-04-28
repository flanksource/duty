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

type OwnedProperty struct {
	types.Property

	CreatedBy   string `json:"created_by,omitempty"`
	CreatorType string `json:"creator_type,omitempty"`
}

type OwnedProperties []*OwnedProperty

func NewOwnedProperties(props types.Properties) OwnedProperties {
	if props == nil {
		return nil
	}

	out := make(OwnedProperties, len(props))
	for i, prop := range props {
		if prop == nil {
			continue
		}
		out[i] = &OwnedProperty{Property: *prop.DeepCopy()}
	}
	return out
}

func (p OwnedProperties) AsProperties() types.Properties {
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

func (m OwnedProperties) MarshalJSON() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	t := ([]*OwnedProperty)(m)
	return json.Marshal(t)
}

func (m *OwnedProperties) UnmarshalJSON(b []byte) error {
	t := []*OwnedProperty{}
	err := json.Unmarshal(b, &t)
	*m = OwnedProperties(t)
	return err
}

func (p OwnedProperties) AsJSON() []byte {
	if len(p) == 0 {
		return []byte("[]")
	}
	data, err := json.Marshal(p)
	if err != nil {
		logger.Errorf("Error marshalling config item properties: %v", err)
	}
	return data
}

func (p OwnedProperties) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return p.AsJSON(), nil
}

func (p *OwnedProperties) Scan(val interface{}) error {
	if val == nil {
		*p = make(OwnedProperties, 0)
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

func (OwnedProperties) GormDataType() string {
	return "config_item_properties"
}

func (OwnedProperties) GormDBDataType(db *gorm.DB, field *schema.Field) string {
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

func (p OwnedProperties) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(p)
	return gorm.Expr("?", data)
}

type UpdateConfigItemPropertiesResult struct {
	Changed    bool
	Properties OwnedProperties
}

// UpdateConfigItemProperties replaces only the properties owned by the given
// creator on a config item. Existing properties from other creators, and legacy
// properties without ownership metadata, are preserved; incoming properties are
// stamped with creator_type/created_by before being merged. Passing an empty
// incoming list removes that creator's owned properties.
func UpdateConfigItemProperties(tx *gorm.DB, configID uuid.UUID, creatorType string, createdBy uuid.UUID, incoming types.Properties) (UpdateConfigItemPropertiesResult, error) {
	incomingJSON := incoming.AsJSON()

	var result struct {
		Changed    bool
		Properties string
	}
	if err := tx.Raw(`SELECT changed, properties FROM update_config_item_properties(@configID, @creatorType, @createdBy, CAST(@incoming AS jsonb))`,
		stdsql.Named("configID", configID),
		stdsql.Named("creatorType", creatorType),
		stdsql.Named("createdBy", createdBy),
		stdsql.Named("incoming", string(incomingJSON)),
	).Scan(&result).Error; err != nil {
		return UpdateConfigItemPropertiesResult{}, err
	}

	var merged OwnedProperties
	if result.Properties != "" {
		if err := json.Unmarshal([]byte(result.Properties), &merged); err != nil {
			return UpdateConfigItemPropertiesResult{}, fmt.Errorf("unmarshal merged properties: %w", err)
		}
	}

	return UpdateConfigItemPropertiesResult{Changed: result.Changed, Properties: merged}, nil
}
