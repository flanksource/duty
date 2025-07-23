package view

import (
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
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
)

// ViewColumnDef defines a column in the view
// +kubebuilder:object:generate=true
// +kubebuilder:validation:XValidation:rule="self.type=='gauge' ? has(self.gauge) : !has(self.gauge)",message="gauge config required when type is gauge, not allowed for other types"
type ViewColumnDef struct {
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

	// For references the column this column is for.
	// Applicable only for type=url.
	//
	// When a column is designated for a different column,
	// it's not rendered on the UI but the designated column uses it to render itself.
	For *string `json:"for,omitempty" yaml:"for,omitempty"`
}

// GaugeThreshold defines a threshold configuration for gauge charts
// +kubebuilder:object:generate=true
type GaugeThreshold struct {
	Value int    `json:"value" yaml:"value"`
	Color string `json:"color" yaml:"color"`
}

// GaugeConfig defines configuration for gauge visualization
// +kubebuilder:object:generate=true
type GaugeConfig struct {
	Min        int              `json:"min,omitempty" yaml:"min,omitempty"`
	Max        int              `json:"max,omitempty" yaml:"max,omitempty"`
	Thresholds []GaugeThreshold `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`
}

type ViewColumnDefList []ViewColumnDef

func (c ViewColumnDefList) SelectColumns() []string {
	output := make([]string, len(c))
	for i, col := range c {
		output[i] = col.Name
	}

	return output
}

func (c ViewColumnDefList) PrimaryKey() []string {
	return lo.Map(lo.Filter(c, func(col ViewColumnDef, _ int) bool {
		return col.PrimaryKey
	}), func(col ViewColumnDef, _ int) string {
		return col.Name
	})
}

func (c ViewColumnDefList) ToColumnTypeMap() map[string]models.ColumnType {
	return lo.SliceToMap(c, func(col ViewColumnDef) (string, models.ColumnType) {
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
