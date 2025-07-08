package view

import (
	"fmt"
	"strings"

	"github.com/flanksource/duty/context"
)

type ViewColumnType string

const (
	ViewColumnTypeString   ViewColumnType = "string"
	ViewColumnTypeNumber   ViewColumnType = "number"
	ViewColumnTypeBoolean  ViewColumnType = "boolean"
	ViewColumnTypeDateTime ViewColumnType = "datetime"
	ViewColumnTypeDuration ViewColumnType = "duration"
	ViewColumnTypeHealth   ViewColumnType = "health"
	ViewColumnTypeStatus   ViewColumnType = "status"
	ViewColumnTypeGauge    ViewColumnType = "gauge"
)

// ViewRow represents a single row of data mapped to view columns
type ViewRow []any

// ViewColumnDef defines a column in the view
// +kubebuilder:object:generate=true
// +kubebuilder:validation:XValidation:rule="self.type=='gauge' ? has(self.gauge) : !has(self.gauge)",message="gauge config required when type is gauge, not allowed for other types"
type ViewColumnDef struct {
	// Name of the column
	Name string `json:"name" yaml:"name"`

	// +kubebuilder:validation:Enum=string;number;boolean;datetime;duration;health;status;gauge
	Type ViewColumnType `json:"type" yaml:"type"`

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

func CreateViewTable(ctx context.Context, tableName string, columns []ViewColumnDef) error {
	if ctx.DB().Migrator().HasTable(tableName) {
		return nil
	}

	var columnDefs []string
	for _, col := range columns {
		colDef := fmt.Sprintf("%s %s", col.Name, getPostgresType(col.Type))
		columnDefs = append(columnDefs, colDef)
	}

	columnDefs = append(columnDefs, "agent_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid")
	columnDefs = append(columnDefs, "is_pushed BOOLEAN DEFAULT FALSE")

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columnDefs, ", "))
	return ctx.DB().Exec(sql).Error
}

func getPostgresType(colType ViewColumnType) string {
	switch colType {
	case ViewColumnTypeString:
		return "TEXT"
	case ViewColumnTypeNumber:
		return "NUMERIC"
	case ViewColumnTypeBoolean:
		return "BOOLEAN"
	case ViewColumnTypeDateTime:
		return "TIMESTAMP WITH TIME ZONE"
	case ViewColumnTypeDuration:
		return "BIGINT"
	case ViewColumnTypeHealth:
		return "TEXT"
	case ViewColumnTypeStatus:
		return "TEXT"
	case ViewColumnTypeGauge:
		return "JSONB"
	default:
		return "TEXT"
	}
}
