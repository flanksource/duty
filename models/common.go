package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SQLite fundamental data types
const (
	SQLiteTypeINTEGER = "INTEGER"
	SQLiteTypeREAL    = "REAL"
	SQLiteTypeTEXT    = "TEXT"
	SQLiteTypeBLOB    = "BLOB"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

func (s Severity) Pretty() api.Text {
	switch s {
	case SeverityCritical:
		return clicky.Text(string(s), "uppercase font-bold text-red-600 bg-red-50")
	case SeverityHigh:
		return clicky.Text(string(s), "uppercase font-bold text-orange-600 bg-orange-50")
	case SeverityMedium:
		return clicky.Text(string(s), "capitalize text-yellow-700 bg-yellow-50")
	case SeverityLow:
		return clicky.Text(string(s), "capitalize text-blue-600 bg-blue-50")
	case SeverityInfo:
		return clicky.Text(string(s), "capitalize text-gray-600")
	default:
		return clicky.Text(string(s), "text-gray-500")
	}
}

type Health string

const (
	HealthHealthy   Health = "healthy"
	HealthUnhealthy Health = "unhealthy"
	HealthUnknown   Health = "unknown"
	HealthWarning   Health = "warning"
)

func (h Health) Pretty() api.Text {
	switch h {
	case HealthHealthy:
		return clicky.Text("✓ ", "text-green-600").Append(string(h), "capitalize text-green-600")
	case HealthUnhealthy:
		return clicky.Text("✗ ", "text-red-600").Append(string(h), "capitalize text-red-600")
	case HealthWarning:
		return clicky.Text("! ", "text-yellow-600").Append(string(h), "capitalize text-yellow-600")
	case HealthUnknown:
		return clicky.Text("? ", "text-gray-500").Append(string(h), "capitalize text-gray-500")
	default:
		return clicky.Text(string(h), "text-gray-500")
	}
}

func WorseHealth(healths ...Health) Health {
	worst := HealthHealthy
	for _, h := range healths {
		switch h {
		case HealthUnhealthy:
			return HealthUnhealthy
		case HealthWarning:
			worst = HealthWarning
		}
	}
	return worst
}

func init() {
	logger.SkipFrameContains = append(logger.SkipFrameContains, "duty/models")
}

// asMap marshals the given struct into a map.
func asMap(t any, removeFields ...string) map[string]any {
	m := make(map[string]any)
	b, _ := json.Marshal(&t)
	if err := json.Unmarshal(b, &m); err != nil {
		return m
	}

	for _, field := range removeFields {
		delete(m, field)
	}

	return m
}

type DBTable interface {
	PK() string
	TableName() string
}

type Deleteable interface {
	Delete(db *gorm.DB) error
}

// TODO: Find a better way to handle this
type ExtendedDBTable interface {
	DBTable
	Value() any
	PKCols() []clause.Column
}

func GetIDs[T DBTable](items ...T) []string {
	var ids []string
	for _, item := range items {
		ids = append(ids, item.PK())
	}
	return ids
}

type Contextable interface {
	Context() map[string]any
}

func ErrorContext(items ...Contextable) []any {
	merged := make(map[string]any)

	for _, item := range items {
		if item == nil {
			continue
		}
		for k, v := range item.Context() {
			merged[k] = v
		}
	}
	var args []any

	for k, v := range merged {
		if v == nil || v == uuid.Nil.String() {
			continue
		}
		args = append(args, k, v)
	}
	return args
}

type LogNameAccessor interface {
	LoggerName() string
}

type NamespaceScopeAccessor interface {
	GetNamespace() string
}

// noopMatcher implements TagsMatchable
type noopMatcher struct {
}

func (t noopMatcher) Has(field string) (exists bool) {
	return false
}

func (t noopMatcher) Get(field string) (value string) {
	return ""
}

func (t noopMatcher) Lookup(field string) (value string, exists bool) {
	return "", false
}

// ABACAttribute is the object passed to casbin for authorization checks.
//
// NOTE: the fields are not a pointer to avoid nil pointer checks in the casbin policy.
type ABACAttribute struct {
	Playbook   Playbook
	Connection Connection
	Component  Component
	Config     ConfigItem
	Check      Check
	Canary     Canary
	View       View
}

type TaggableModel interface {
	GetTags() map[string]string
}

type LabelableModel interface {
	GetLabels() map[string]string
}

type ColumnType string

const (
	ColumnTypeBoolean  ColumnType = "boolean"
	ColumnTypeDateTime ColumnType = "datetime"
	ColumnTypeDecimal  ColumnType = "decimal"
	ColumnTypeDuration ColumnType = "duration"
	ColumnTypeInteger  ColumnType = "integer"
	ColumnTypeJSONB    ColumnType = "jsonb"
	ColumnTypeString   ColumnType = "string"
)

func (t ColumnType) SQLiteType() string {
	switch t {
	case ColumnTypeBoolean:
		return SQLiteTypeINTEGER
	case ColumnTypeDateTime:
		return SQLiteTypeTEXT
	case ColumnTypeDecimal:
		return SQLiteTypeREAL
	case ColumnTypeDuration:
		return SQLiteTypeINTEGER
	case ColumnTypeInteger:
		return SQLiteTypeINTEGER
	case ColumnTypeJSONB:
		return SQLiteTypeBLOB
	case ColumnTypeString:
		return SQLiteTypeTEXT
	default:
		return SQLiteTypeTEXT
	}
}

// ConvertRowToNativeTypes converts a database row to native go types
func ConvertRowToNativeTypes(row map[string]any, columnDef map[string]ColumnType) (map[string]any, map[string]string) {
	// keep track of conversion error per column
	invalidTypesPerColumn := make(map[string]string)

	for colName, value := range row {
		colType, ok := columnDef[colName]
		if !ok {
			// These could be columns that the system generates.
			// They are not in the user specified columnDef.
			switch colName {
			case "agent_id":
				// Do nothing

			case "__row__attributes":
				colType = ColumnTypeJSONB

			default:
				continue
			}

		}

		switch colType {
		case ColumnTypeJSONB:
			if raw, ok := value.([]uint8); ok {
				row[colName] = json.RawMessage(raw)
			}

		case ColumnTypeDuration:
			switch v := value.(type) {
			case int:
				row[colName] = time.Duration(v)
			case int32:
				row[colName] = time.Duration(v)
			case int64:
				row[colName] = time.Duration(v)
			case float64:
				row[colName] = time.Duration(int64(v))
			case string:
				if parsed, err := time.ParseDuration(v); err != nil {
					if _, exists := invalidTypesPerColumn[colName]; !exists {
						invalidTypesPerColumn[colName] = fmt.Sprintf("failed to parse duration (value: %v)", v)
					}
					row[colName] = nil
				} else {
					row[colName] = parsed
				}
			case nil:
				row[colName] = nil
			default:
				if _, exists := invalidTypesPerColumn[colName]; !exists {
					invalidTypesPerColumn[colName] = fmt.Sprintf("invalid type %T", v)
				}
				row[colName] = nil
			}

		case ColumnTypeDateTime:
			switch v := value.(type) {
			case time.Time:
				row[colName] = v
			case string:
				if parsed, err := time.Parse(time.RFC3339, v); err != nil {
					if _, exists := invalidTypesPerColumn[colName]; !exists {
						invalidTypesPerColumn[colName] = fmt.Sprintf("failed to parse datetime (value: %v)", v)
					}
					row[colName] = nil
				} else {
					row[colName] = parsed
				}
			case nil:
				row[colName] = nil
			default:
				if _, exists := invalidTypesPerColumn[colName]; !exists {
					invalidTypesPerColumn[colName] = fmt.Sprintf("invalid type %T", v)
				}
				row[colName] = nil
			}

		case ColumnTypeString:
			if value == nil {
				row[colName] = ""
			} else {
				row[colName] = fmt.Sprintf("%v", value)
			}

		case ColumnTypeInteger:
			if value == nil {
				row[colName] = 0
			}

		case ColumnTypeDecimal:
			if value == nil {
				row[colName] = float64(0)
			}

		case ColumnTypeBoolean:
			switch v := value.(type) {
			case bool:
				row[colName] = v
			case string:
				if parsed, err := strconv.ParseBool(v); err != nil {
					if _, exists := invalidTypesPerColumn[colName]; !exists {
						invalidTypesPerColumn[colName] = fmt.Sprintf("failed to parse boolean (value: %v)", v)
					}
					row[colName] = false
				} else {
					row[colName] = parsed
				}
			case int, int32, int64:
				row[colName] = v != 0
			case nil:
				row[colName] = false
			default:
				if _, exists := invalidTypesPerColumn[colName]; !exists {
					invalidTypesPerColumn[colName] = fmt.Sprintf("invalid boolean type %T", v)
				}
				row[colName] = false
			}

		default:
			// do nothing
		}
	}

	return row, invalidTypesPerColumn
}
