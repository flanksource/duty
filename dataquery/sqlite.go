package dataquery

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/flanksource/commons/collections/set"
	"github.com/glebarez/sqlite"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// The number of rows to sample to infer column types
const defaultSampleSize = 150

// QueryResultSet contains the query name and the results
type QueryResultSet struct {
	Name    string
	Results []QueryResultRow

	// Map column name to column type
	ColumnDefs map[string]models.ColumnType
}

func DBFromResultsets(ctx context.Context, resultsets []QueryResultSet) (context.Context, func() error, error) {
	if len(resultsets) == 0 {
		return ctx, nil, fmt.Errorf("resultsets cannot be empty")
	}

	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to create in-memory SQLite database: %w", err)
	}

	sqlDB, err := sqliteDB.DB()
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqliteCtx := ctx.WithDB(sqliteDB, nil)

	// Create tables for each result set and insert the rows
	for _, resultSet := range resultsets {
		if err := resultSet.CreateDBTable(sqliteCtx); err != nil {
			return ctx, sqlDB.Close, fmt.Errorf("failed to create table for result set '%s': %w", resultSet.Name, err)
		}

		if err := resultSet.InsertToDB(sqliteCtx); err != nil {
			return ctx, sqlDB.Close, fmt.Errorf("failed to insert data into table '%s': %w", resultSet.Name, err)
		}
	}

	return sqliteCtx, sqlDB.Close, nil
}

// InferColumnTypes analyzes the first few rows to determine the most appropriate column types
func InferColumnTypes(rows []QueryResultRow) map[string]string {
	if len(rows) == 0 {
		return map[string]string{}
	}

	// Track types seen for each column from the first N rows
	columnTypeSets := make(map[string]set.Set[string])
	for i, row := range rows {
		if i >= defaultSampleSize {
			break
		}

		for col, val := range row {
			if columnTypeSets[col] == nil {
				columnTypeSets[col] = set.New[string]()
			}

			if val != nil {
				sqliteType := goTypeToSQLiteType(val)
				columnTypeSets[col].Add(sqliteType)
			}
		}
	}

	// Determine the most appropriate type for each column
	columnTypes := make(map[string]string)
	for col, typeSet := range columnTypeSets {
		columnTypes[col] = inferBestColumnTypeFromSet(typeSet)
	}

	return columnTypes
}

// inferBestColumnTypeFromSet determines the most appropriate SQLite type from a set of observed types
func inferBestColumnTypeFromSet(typeSet set.Set[string]) string {
	if len(typeSet) == 0 {
		return models.SQLiteTypeTEXT
	}

	if typeSet.Contains(models.SQLiteTypeBLOB) {
		return models.SQLiteTypeBLOB
	}

	if typeSet.Contains(models.SQLiteTypeTEXT) {
		return models.SQLiteTypeTEXT
	}

	if typeSet.Contains(models.SQLiteTypeREAL) {
		return models.SQLiteTypeREAL
	}

	return models.SQLiteTypeINTEGER
}

// goTypeToSQLiteType converts a Go value to SQLite column type
func goTypeToSQLiteType(value any) string {
	if value == nil {
		return models.SQLiteTypeTEXT
	}

	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return models.SQLiteTypeINTEGER
	case float32, float64:
		return models.SQLiteTypeREAL
	case bool:
		return models.SQLiteTypeINTEGER // SQLite stores booleans as integers
	case time.Time:
		return models.SQLiteTypeTEXT // Store as ISO string
	case string:
		return models.SQLiteTypeTEXT
	case []byte, json.RawMessage, types.JSON:
		return models.SQLiteTypeBLOB
	case types.JSONMap, types.JSONStringMap, map[string]any, map[string]string:
		return models.SQLiteTypeBLOB
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return models.SQLiteTypeINTEGER
		case reflect.Float32, reflect.Float64:
			return models.SQLiteTypeREAL
		case reflect.Bool:
			return models.SQLiteTypeINTEGER
		case reflect.Map, reflect.Slice:
			return models.SQLiteTypeBLOB
		default:
			return models.SQLiteTypeTEXT
		}
	}
}

// CreateDBTable creates a SQLite table based on the result set schema
func (resultSet QueryResultSet) CreateDBTable(ctx context.Context) error {
	if len(resultSet.ColumnDefs) == 0 && len(resultSet.Results) == 0 {
		return fmt.Errorf("cannot create SQLite table from empty result set without column definitions")
	}

	var columnTypes map[string]string
	if len(resultSet.ColumnDefs) != 0 {
		columnTypes = make(map[string]string)
		for columnName, columnType := range resultSet.ColumnDefs {
			columnTypes[columnName] = columnType.SQLiteType()
		}
	} else {
		// Infer column types from data when results are available
		columnTypes = InferColumnTypes(resultSet.Results)
	}

	var columnDefs []string
	for columnName, columnType := range columnTypes {
		columnDefs = append(columnDefs, fmt.Sprintf(`"%s" %s`, columnName, columnType))
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
