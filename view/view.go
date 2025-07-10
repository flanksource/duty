package view

import (
	"strings"

	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
)

type ColumnType string

const (
	ColumnTypeString   ColumnType = "string"
	ColumnTypeNumber   ColumnType = "number"
	ColumnTypeBoolean  ColumnType = "boolean"
	ColumnTypeDateTime ColumnType = "datetime"
	ColumnTypeDuration ColumnType = "duration"
	ColumnTypeHealth   ColumnType = "health"
	ColumnTypeStatus   ColumnType = "status"
	ColumnTypeGauge    ColumnType = "gauge"
)

// ViewColumnDef defines a column in the view
// +kubebuilder:object:generate=true
// +kubebuilder:validation:XValidation:rule="self.type=='gauge' ? has(self.gauge) : !has(self.gauge)",message="gauge config required when type is gauge, not allowed for other types"
type ViewColumnDef struct {
	// Name of the column
	Name string `json:"name" yaml:"name"`

	// PrimaryKey indicates if the column is a primary key
	PrimaryKey bool `json:"primaryKey,omitempty" yaml:"primaryKey,omitempty"`

	// +kubebuilder:validation:Enum=string;number;boolean;datetime;duration;health;status;gauge
	Type ColumnType `json:"type" yaml:"type"`

	// Description of the column
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Configuration for gauge visualization
	Gauge *GaugeConfig `json:"gauge,omitempty" yaml:"gauge,omitempty"`
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
		// The column name we receive from postgres is always in lowercase.
		name := strings.ToLower(col.Name)

		switch col.Type {
		case ColumnTypeNumber:
			return name, models.ColumnTypeNumber
		case ColumnTypeBoolean:
			return name, models.ColumnTypeBoolean
		case ColumnTypeDateTime:
			return name, models.ColumnTypeDateTime
		case ColumnTypeDuration:
			return name, models.ColumnTypeDuration
		case ColumnTypeGauge:
			return name, models.ColumnTypeJSONB
		default:
			return name, models.ColumnTypeString
		}
	})
}
