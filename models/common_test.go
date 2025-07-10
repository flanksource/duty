package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func TestConvertRowToNativeTypes(t *testing.T) {
	tests := []struct {
		name           string
		row            map[string]any
		columnDef      map[string]ColumnType
		expectedRow    map[string]any
		expectedErrors map[string]string
	}{
		{
			name:           "string_nil_to_empty",
			row:            map[string]any{"col1": nil},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeString},
			expectedRow:    map[string]any{"col1": ""},
			expectedErrors: map[string]string{},
		},
		{
			name:           "string_int_to_string",
			row:            map[string]any{"col1": 123},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeString},
			expectedRow:    map[string]any{"col1": "123"},
			expectedErrors: map[string]string{},
		},
		{
			name:           "string_bool_to_string",
			row:            map[string]any{"col1": true},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeString},
			expectedRow:    map[string]any{"col1": "true"},
			expectedErrors: map[string]string{},
		},
		// Integer column type tests
		{
			name:           "integer_nil_to_zero",
			row:            map[string]any{"col1": nil},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeInteger},
			expectedRow:    map[string]any{"col1": 0},
			expectedErrors: map[string]string{},
		},
		// Decimal column type tests
		{
			name:           "decimal_nil_to_zero",
			row:            map[string]any{"col1": nil},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDecimal},
			expectedRow:    map[string]any{"col1": float64(0)},
			expectedErrors: map[string]string{},
		},
		// Boolean column type tests
		{
			name:           "boolean_true",
			row:            map[string]any{"col1": true},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": true},
			expectedErrors: map[string]string{},
		},
		{
			name:           "boolean_false",
			row:            map[string]any{"col1": false},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": false},
			expectedErrors: map[string]string{},
		},
		{
			name:           "boolean_string_true",
			row:            map[string]any{"col1": "true"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": true},
			expectedErrors: map[string]string{},
		},
		{
			name:           "boolean_int_nonzero",
			row:            map[string]any{"col1": 1},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": true},
			expectedErrors: map[string]string{},
		},
		{
			name:           "boolean_int_zero",
			row:            map[string]any{"col1": 0},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": false},
			expectedErrors: map[string]string{},
		},
		{
			name:           "boolean_nil_to_false",
			row:            map[string]any{"col1": nil},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": false},
			expectedErrors: map[string]string{},
		},
		{
			name:           "boolean_invalid_string",
			row:            map[string]any{"col1": "invalid"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": false},
			expectedErrors: map[string]string{"col1": "failed to parse boolean (value: invalid)"},
		},
		{
			name:           "boolean_invalid_type",
			row:            map[string]any{"col1": []string{"invalid"}},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeBoolean},
			expectedRow:    map[string]any{"col1": false},
			expectedErrors: map[string]string{"col1": "invalid boolean type []string"},
		},
		// DateTime column type tests
		{
			name:           "datetime_time_preserved",
			row:            map[string]any{"col1": time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDateTime},
			expectedRow:    map[string]any{"col1": time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "datetime_rfc3339_string",
			row:            map[string]any{"col1": "2023-01-01T12:00:00Z"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDateTime},
			expectedRow:    map[string]any{"col1": time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "datetime_nil",
			row:            map[string]any{"col1": nil},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDateTime},
			expectedRow:    map[string]any{"col1": nil},
			expectedErrors: map[string]string{},
		},
		{
			name:           "datetime_invalid_string",
			row:            map[string]any{"col1": "invalid-date"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDateTime},
			expectedRow:    map[string]any{"col1": nil},
			expectedErrors: map[string]string{"col1": "failed to parse datetime (value: invalid-date)"},
		},
		{
			name:           "datetime_invalid_type",
			row:            map[string]any{"col1": 123},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDateTime},
			expectedRow:    map[string]any{"col1": nil},
			expectedErrors: map[string]string{"col1": "invalid type int"},
		},
		// Duration column type tests
		{
			name:           "duration_int",
			row:            map[string]any{"col1": 60},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": time.Duration(60)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "duration_int32",
			row:            map[string]any{"col1": int32(60)},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": time.Duration(60)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "duration_int64",
			row:            map[string]any{"col1": int64(60)},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": time.Duration(60)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "duration_float64",
			row:            map[string]any{"col1": float64(60.5)},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": time.Duration(60)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "duration_string",
			row:            map[string]any{"col1": "5m"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": 5 * time.Minute},
			expectedErrors: map[string]string{},
		},
		{
			name:           "duration_nil",
			row:            map[string]any{"col1": nil},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": nil},
			expectedErrors: map[string]string{},
		},
		{
			name:           "duration_invalid_string",
			row:            map[string]any{"col1": "invalid-duration"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": nil},
			expectedErrors: map[string]string{"col1": "failed to parse duration (value: invalid-duration)"},
		},
		{
			name:           "duration_invalid_type",
			row:            map[string]any{"col1": []string{"invalid"}},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeDuration},
			expectedRow:    map[string]any{"col1": nil},
			expectedErrors: map[string]string{"col1": "invalid type []string"},
		},
		// JSONB column type tests
		{
			name:           "jsonb_uint8_slice",
			row:            map[string]any{"col1": []uint8(`{"key": "value"}`)},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeJSONB},
			expectedRow:    map[string]any{"col1": json.RawMessage(`{"key": "value"}`)},
			expectedErrors: map[string]string{},
		},
		{
			name:           "jsonb_other_type",
			row:            map[string]any{"col1": "string"},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeJSONB},
			expectedRow:    map[string]any{"col1": "string"},
			expectedErrors: map[string]string{},
		},
		// Multiple columns with errors
		{
			name: "multiple_column_errors",
			row: map[string]any{
				"col1": "invalid-bool",
				"col2": "invalid-date",
				"col3": "invalid-duration",
			},
			columnDef: map[string]ColumnType{
				"col1": ColumnTypeBoolean,
				"col2": ColumnTypeDateTime,
				"col3": ColumnTypeDuration,
			},
			expectedRow: map[string]any{
				"col1": false,
				"col2": nil,
				"col3": nil,
			},
			expectedErrors: map[string]string{
				"col1": "failed to parse boolean (value: invalid-bool)",
				"col2": "failed to parse datetime (value: invalid-date)",
				"col3": "failed to parse duration (value: invalid-duration)",
			},
		},
		// Unknown column types
		{
			name: "ignore_unknown_columns",
			row: map[string]any{
				"col1": "value1",
				"col2": "value2",
			},
			columnDef:      map[string]ColumnType{"col1": ColumnTypeString},
			expectedRow:    map[string]any{"col1": "value1", "col2": "value2"},
			expectedErrors: map[string]string{},
		},
		{
			name:           "unknown_column_type",
			row:            map[string]any{"col1": "value1"},
			columnDef:      map[string]ColumnType{"col1": ColumnType("unknown")},
			expectedRow:    map[string]any{"col1": "value1"},
			expectedErrors: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			result, errors := ConvertRowToNativeTypes(tt.row, tt.columnDef)
			g.Expect(result).To(gomega.Equal(tt.expectedRow))
			g.Expect(errors).To(gomega.Equal(tt.expectedErrors))
		})
	}
}
