package view

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

type ColumnType string

const (
	ColumnTypeBoolean   ColumnType = "boolean"
	ColumnTypeBytes     ColumnType = "bytes"
	ColumnTypeDateTime  ColumnType = "datetime"
	ColumnTypeDecimal   ColumnType = "decimal"
	ColumnTypeDuration  ColumnType = "duration"
	ColumnTypeGauge     ColumnType = "gauge"
	ColumnTypeHealth    ColumnType = "health"
	ColumnTypeMillicore ColumnType = "millicore"
	ColumnTypeNumber    ColumnType = "number"
	ColumnTypeStatus    ColumnType = "status"
	ColumnTypeString    ColumnType = "string"
	ColumnTypeURL       ColumnType = "url"

	// reserved type for internal use.
	// Stores properties for all the columns in a row.
	ColumnTypeAttributes ColumnType = "row_attributes"
)

// ColumnDef defines a column in the view
// +kubebuilder:object:generate=true
// +kubebuilder:validation:XValidation:rule="self.type=='gauge' ? has(self.gauge) : !has(self.gauge)",message="gauge config required when type is gauge, not allowed for other types"
type ColumnDef struct {
	// Name of the column
	Name string `json:"name" yaml:"name"`

	// PrimaryKey indicates if the column is a primary key
	PrimaryKey bool `json:"primaryKey,omitempty" yaml:"primaryKey,omitempty"`

	// +kubebuilder:validation:Enum=string;number;boolean;datetime;duration;health;status;gauge;bytes;decimal;millicore;url
	Type ColumnType `json:"type" yaml:"type"`

	// Description of the column
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Hidden indicates if the column should be hidden from view
	Hidden bool `json:"hidden,omitempty" yaml:"hidden,omitempty"`

	// Configuration for gauge visualization
	Gauge *GaugeConfig `json:"gauge,omitempty" yaml:"gauge,omitempty"`

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
}

func (c *ColumnDef) HasAttributes() bool {
	return c.URL != nil ||
		(c.Gauge != nil && (c.Gauge.Max != "" || c.Gauge.Min != ""))
}

// +kubebuilder:object:generate=true
type ColumnURL struct {
	// ID of the config to link to.
	Config string `json:"config,omitempty" template:"true"`

	// Link to a view.
	View *ViewURLRef `json:"view,omitempty" template:"true"`
}

func (colURL ColumnURL) Eval(env map[string]any) (any, error) {
	c := colURL.DeepCopy()

	if c.Config != "" {
		configID, err := types.CelExpression(c.Config).Eval(env)
		if err != nil {
			return nil, err
		}

		return fmt.Sprintf("/catalog/%s", configID), nil
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

		return fmt.Sprintf("/views/%s/%s", c.View.Namespace, c.View.Name), nil
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
	return lo.Map(lo.Filter(c, func(col ColumnDef, _ int) bool {
		return col.PrimaryKey
	}), func(col ColumnDef, _ int) string {
		return col.Name
	})
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
		case ColumnTypeURL:
			return col.Name, models.ColumnTypeString
		default:
			return col.Name, models.ColumnTypeString
		}
	})
}
