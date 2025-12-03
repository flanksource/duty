package view

import (
	"fmt"
	"net/url"

	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

type ColumnType string

const (
	ColumnTypeBoolean    ColumnType = "boolean"
	ColumnTypeBytes      ColumnType = "bytes"
	ColumnTypeConfigItem ColumnType = "config_item"
	ColumnTypeDateTime   ColumnType = "datetime"
	ColumnTypeDecimal    ColumnType = "decimal"
	ColumnTypeDuration   ColumnType = "duration"
	ColumnTypeGauge      ColumnType = "gauge"
	ColumnTypeHealth     ColumnType = "health"
	ColumnTypeMillicore  ColumnType = "millicore"
	ColumnTypeNumber     ColumnType = "number"
	ColumnTypeStatus     ColumnType = "status"
	ColumnTypeString     ColumnType = "string"
	ColumnTypeURL        ColumnType = "url"
	ColumnTypeBadge      ColumnType = "badge"
	ColumnTypeLabels     ColumnType = "labels"

	// reserved type for internal use.
	// Stores properties for all the columns in a row.
	ColumnTypeAttributes ColumnType = "row_attributes"

	// reserved type for internal use.
	// Stores scope UUIDs for row-level access control.
	ColumnTypeGrants ColumnType = "grants"
)

// CardPosition defines predefined card rendering styles
type CardPosition string

const (
	CardPositionTitle    CardPosition = "title"
	CardPositionSubtitle CardPosition = "subtitle"
	CardPositionBody     CardPosition = "body"
	CardPositionFooter   CardPosition = "footer"

	// Show on the header after subtitle
	CardPositionDeck CardPosition = "deck"

	// Show on the header right side
	CardPositionHeaderRight CardPosition = "headerRight"
)

// CardConfig defines card layout configuration
// +kubebuilder:object:generate=true
type CardConfig struct {
	// Position defines where the field is displayed on the card
	// +kubebuilder:validation:Enum=title;subtitle;deck;body;footer;headerRight
	Position string `json:"position,omitempty" yaml:"position,omitempty"`

	// UseForAccent indicates if this column's value should be used for the accent color
	UseForAccent bool `json:"useForAccent,omitempty" yaml:"useForAccent,omitempty"`
}

// ColumnDef defines a column in the view
// +kubebuilder:object:generate=true
// +kubebuilder:validation:XValidation:rule="self.type=='gauge' ? has(self.gauge) : !has(self.gauge)",message="gauge config required when type is gauge, not allowed for other types"
type ColumnDef struct {
	// Name of the column
	Name string `json:"name" yaml:"name"`

	// PrimaryKey indicates if the column is a primary key
	PrimaryKey bool `json:"primaryKey,omitempty" yaml:"primaryKey,omitempty"`

	// +kubebuilder:validation:Enum=string;number;boolean;datetime;duration;health;status;gauge;bytes;decimal;millicore;url;badge;config_item;labels
	Type ColumnType `json:"type" yaml:"type"`

	// Description of the column
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Hidden indicates if the column should be hidden from view
	Hidden bool `json:"hidden,omitempty" yaml:"hidden,omitempty"`

	// Configuration for gauge visualization
	Gauge *GaugeConfig `json:"gauge,omitempty" yaml:"gauge,omitempty"`

	// Configuration for config item columns
	ConfigItem *ConfigItemColumnType `json:"configItem,omitempty"`

	// Icon to use for the column.
	//
	// Supports mapping to a row value.
	// Example: icon: row.health
	Icon *string `json:"icon,omitempty" yaml:"icon,omitempty"`

	// Deprecated: Use properties instead. Example: URL
	//
	// For references the column this column is for.
	// Applicable only for type=url.
	//
	// When a column is designated for a different column,
	// it's not rendered on the UI but the designated column uses it to render itself.
	For *string `json:"for,omitempty" yaml:"for,omitempty"`

	// Enable filters in the UI
	Filter *ColumnFilter `json:"filter,omitempty" yaml:"filter,omitempty"`

	// Link to various mission control components.
	URL *ColumnURL `json:"url,omitempty" yaml:"url,omitempty"`

	// Unit of the column
	Unit string `json:"unit,omitempty" yaml:"unit,omitempty"`

	// Card defines the card layout configuration for the field
	Card *CardConfig `json:"card,omitempty" yaml:"card,omitempty"`

	// Deprecated: Use Card instead
	// +kubebuilder:validation:Enum=title;subtitle;deck;body;footer;headerRight
	// CardPosition defines the visual presentation style for the card field
	CardPosition CardPosition `json:"cardPosition,omitempty"`
}

// +kubebuilder:object:generate=true
type ColumnURL struct {
	// Cel expression that evaluates to the id of the config or search query.
	// Example:
	// - uuid: row.id
	// - search term:  f("name=$(tags.namespace) tags.cluster=$(tags.cluster) type=Kubernetes::Namespace", row)
	Config types.CelExpression `json:"config,omitempty" template:"true"`

	// Template a custom URL using Go template.
	Template types.GoTemplate `json:"template,omitempty" yaml:"template,omitempty"`

	// Link to a view.
	View *ViewURLRef `json:"view,omitempty" template:"true"`
}

func (colURL ColumnURL) Eval(ctx context.Context, env map[string]any) (any, error) {
	c := colURL.DeepCopy()

	if c.Config != "" {
		config, err := types.CelExpression(c.Config).Eval(env)
		if err != nil {
			return nil, err
		}

		if _, err := uuid.Parse(config); err == nil {
			return fmt.Sprintf("/catalog/%s", config), nil
		} else if len(config) == 0 {
			ctx.Logger.V(6).Infof("ColumnURL.Config evaluated to empty string '%s'", c.Config)
		} else {
			resourceSelector := types.ResourceSelector{Search: config}
			configIDs, err := query.FindConfigIDsByResourceSelector(ctx, 1, resourceSelector)
			if err != nil {
				return nil, fmt.Errorf("failed to execute resource selector '%s': %w", config, err)
			}

			if len(configIDs) > 0 {
				config = configIDs[0].String()
				return fmt.Sprintf("/catalog/%s", config), nil
			} else {
				ctx.Logger.V(6).Infof("no config found for ColumnURL.Config search term '%s'", config)
			}
		}
	}

	if c.View != nil {
		if c.View.Namespace != "" {
			n, err := types.CelExpression(c.View.Namespace).Eval(env)
			if err != nil {
				return nil, err
			}
			c.View.Namespace = n
		}

		if c.View.Name != "" {
			n, err := types.CelExpression(c.View.Name).Eval(env)
			if err != nil {
				return nil, err
			}
			c.View.Name = n
		}

		for k, v := range c.View.Filter {
			vv, err := types.CelExpression(v).Eval(env)
			if err != nil {
				return nil, err
			}
			c.View.Filter[k] = vv
		}

		baseURL := fmt.Sprintf("/view/%s/%s", c.View.Namespace, c.View.Name)
		if c.View.Namespace == "" {
			baseURL = fmt.Sprintf("/view/%s", c.View.Name)
		}

		if len(c.View.Filter) > 0 {
			params := url.Values{}
			for k, v := range c.View.Filter {
				params.Set(k, fmt.Sprintf("%v", v))
			}
			return fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil
		}

		return baseURL, nil
	}

	if c.Template != "" {
		custom, err := ctx.RunTemplate(gomplate.Template{Template: string(c.Template)}, env)
		if err != nil {
			return nil, err
		}

		return custom, nil
	}

	return nil, nil
}

// +kubebuilder:object:generate=true
type ViewURLRef struct {
	Namespace string            `json:"namespace,omitempty" template:"true"`
	Name      string            `json:"name,omitempty" template:"true"`
	Filter    map[string]string `json:"filter,omitempty" template:"true"`
}

type ColumnFilterType string

const (
	ColumnFilterTypeMultiSelect ColumnFilterType = "multiselect"
)

type ColumnFilter struct {
	Type ColumnFilterType `json:"type" yaml:"type"`
}

// GaugeThreshold defines a threshold configuration for gauge charts
// +kubebuilder:object:generate=true
type GaugeThreshold struct {
	// Deprecated: Use Percent instead
	// +kubebuilder:validation:Optional
	Value int `json:"value,omitempty" yaml:"value,omitempty"`

	// Percent is the percentage value of the threshold
	Percent int `json:"percent" yaml:"percent"`

	// Color is the color of the threshold
	Color string `json:"color" yaml:"color"`
}

// GaugeConfig defines configuration for gauge visualization
// +kubebuilder:object:generate=true
type GaugeConfig struct {
	Max        string           `json:"max,omitempty" yaml:"max,omitempty"`
	Min        string           `json:"min,omitempty" yaml:"min,omitempty"`
	Precision  int              `json:"precision,omitempty" yaml:"precision,omitempty"`
	Thresholds []GaugeThreshold `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`
}

type ConfigItemColumnType struct {
	// IDField which field, from the merged query result, to use as the config item ID
	//
	// If not specified, defaults to "id"
	IDField string `json:"idField,omitempty"`
}

type ViewColumnDefList []ColumnDef

func (c ViewColumnDefList) SelectColumns() []string {
	output := make([]string, len(c))
	for i, col := range c {
		output[i] = col.Name
	}

	return output
}

func (c ViewColumnDefList) QuotedColumns() []string {
	output := make([]string, len(c))
	for i, col := range c {
		output[i] = pq.QuoteIdentifier(col.Name)
	}
	return output
}

func (c ViewColumnDefList) PrimaryKey() []string {
	userDefinedPKs := lo.Map(lo.Filter(c, func(col ColumnDef, _ int) bool {
		return col.PrimaryKey
	}), func(col ColumnDef, _ int) string {
		return col.Name
	})
	mustHavePKs := []string{"request_fingerprint"}
	pkColumns := append(userDefinedPKs, mustHavePKs...)
	return pkColumns
}

func (c ViewColumnDefList) ToColumnTypeMap() map[string]models.ColumnType {
	return lo.SliceToMap(c, func(col ColumnDef) (string, models.ColumnType) {
		switch col.Type {
		case ColumnTypeNumber:
			return col.Name, models.ColumnTypeInteger
		case ColumnTypeDecimal:
			return col.Name, models.ColumnTypeDecimal
		case ColumnTypeBytes:
			return col.Name, models.ColumnTypeString
		case ColumnTypeMillicore:
			return col.Name, models.ColumnTypeString
		case ColumnTypeBoolean:
			return col.Name, models.ColumnTypeBoolean
		case ColumnTypeDateTime:
			return col.Name, models.ColumnTypeDateTime
		case ColumnTypeDuration:
			return col.Name, models.ColumnTypeDuration
		case ColumnTypeGauge:
			return col.Name, models.ColumnTypeJSONB
		case ColumnTypeLabels:
			return col.Name, models.ColumnTypeJSONB
		case ColumnTypeURL:
			return col.Name, models.ColumnTypeString
		case ColumnTypeConfigItem:
			return col.Name, models.ColumnTypeString
		default:
			return col.Name, models.ColumnTypeString
		}
	})
}
