package dataquery

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

// QueryResultSet contains the query name and the results
type QueryResultSet struct {
	Name       string
	PrimaryKey []string
	Results    []QueryResultRow
}

// InferColumnTypes analyzes the first row to determine column types
func InferColumnTypes(rows []QueryResultRow) map[string]string {
	if len(rows) == 0 {
		return map[string]string{}
	}

	columnTypes := make(map[string]string)
	firstRow := rows[0]
	for col := range firstRow {
		columnTypes[col] = inferColumnType(firstRow, col)
	}

	return columnTypes
}

// inferColumnType determines the SQLite type for a specific column
func inferColumnType(row QueryResultRow, columnName string) string {
	if val, exists := row[columnName]; exists {
		return goTypeToSQLiteType(val)
	}

	return "TEXT"
}

// goTypeToSQLiteType converts a Go value to SQLite column type
func goTypeToSQLiteType(value any) string {
	if value == nil {
		return "TEXT"
	}

	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "INTEGER"
	case float32, float64:
		return "REAL"
	case bool:
		return "INTEGER" // SQLite stores booleans as integers
	case time.Time:
		return "TEXT" // Store as ISO string
	case string:
		return "TEXT"
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "INTEGER"
		case reflect.Float32, reflect.Float64:
			return "REAL"
		case reflect.Bool:
			return "INTEGER"
		default:
			return "TEXT"
		}
	}
}

// CreateDBTable creates a SQLite table based on the result set schema
func (resultSet QueryResultSet) CreateDBTable(ctx context.Context) error {
	if len(resultSet.Results) == 0 {
		return fmt.Errorf("cannot create table from empty result set")
	}

	columnTypes := InferColumnTypes(resultSet.Results)

	var columnDefs []string
	for columnName, columnType := range columnTypes {
		columnDefs = append(columnDefs, fmt.Sprintf(`"%s" %s`, columnName, columnType))
	}

	if len(resultSet.PrimaryKey) > 0 {
		var primaryKeys []string
		for _, pk := range resultSet.PrimaryKey {
			primaryKeys = append(primaryKeys, fmt.Sprintf(`"%s"`, pk))
		}
		columnDefs = append(columnDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	createTableSQL := fmt.Sprintf(`CREATE TABLE "%s" (%s)`,
		resultSet.Name,
		strings.Join(columnDefs, ", "))

	if err := ctx.DB().Exec(createTableSQL).Error; err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// InsertToDB inserts QueryResultRow data into the specified table
func (rs QueryResultSet) InsertToDB(ctx context.Context) error {
	if len(rs.Results) == 0 {
		return nil
	}

	toInsert := make([]map[string]any, 0, len(rs.Results))

	for _, row := range rs.Results {
		// .Create inserts additional fields to the row (example: a new @id field)
		// So we need to clone the row to avoid modifying the original row
		clone := maps.Clone(row)

		// NOTE: Must typecast QueryResultRow to map[string]any else gorm panics
		clonedMap := map[string]any(clone)

		// Convert complex types to appropriate types.* equivalents
		if err := normalizeRow(clonedMap); err != nil {
			return fmt.Errorf("failed to convert complex types for table '%s': %w", rs.Name, err)
		}

		toInsert = append(toInsert, clonedMap)
	}

	result := ctx.DB().Table(rs.Name).CreateInBatches(toInsert, 100)
	if result.Error != nil {
		return fmt.Errorf("failed to insert row into table '%s': %w", rs.Name, result.Error)
	}

	return nil
}

func normalizeRow(row map[string]any) error {
	for k, v := range row {
		nv, err := normalizeValue(v)
		if err != nil {
			return fmt.Errorf("column %q: %w", k, err)
		}

		row[k] = nv
	}

	return nil
}

func normalizeValue(v any) (any, error) {
	switch x := v.(type) {
	case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64, string, []byte, time.Time, *time.Time, uuid.UUID, *uuid.UUID, types.JSON, types.JSONMap, types.JSONStringMap:
		return x, nil // already good

	case json.RawMessage:
		return types.JSON(x), nil

	default: // fallback: encode as JSONB
		b, err := json.Marshal(x)
		if err != nil {
			return nil, err
		}
		return types.JSON(b), nil
	}
}
