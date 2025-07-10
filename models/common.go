package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Health string

const (
	HealthHealthy   Health = "healthy"
	HealthUnhealthy Health = "unhealthy"
	HealthUnknown   Health = "unknown"
	HealthWarning   Health = "warning"
)

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

// ABACAttribute is the object passed to casbin for authorization checks.
//
// NOTE: the fields are not a pointer to avoid nil pointer checks in the casbin policy.
type ABACAttribute struct {
	Playbook   Playbook
	Connection Connection
	Component  Component
	Config     ConfigItem
	Check      Check
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

// ConvertViewRecordsToNativeTypes converts view cell to native go types
func ConvertViewRecordsToNativeTypes(row map[string]any, columnDef map[string]ColumnType) (map[string]any, map[string][]string) {
	// keep track of all the invalid types encountered per column
	invalidTypesPerColumn := make(map[string][]string)

	for colName, value := range row {
		colType, ok := columnDef[colName]
		if !ok {
			continue
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
					invalidTypesPerColumn[colName] = append(invalidTypesPerColumn[colName], fmt.Sprintf("failed to parse duration (value: %v): %v", v, err))
					row[colName] = nil
				} else {
					row[colName] = parsed
				}
			case nil:
				row[colName] = nil
			default:
				invalidTypesPerColumn[colName] = append(invalidTypesPerColumn[colName], fmt.Sprintf("%T", v))
				row[colName] = nil
			}

		case ColumnTypeDateTime:
			switch v := value.(type) {
			case time.Time:
				row[colName] = v
			case string:
				if parsed, err := time.Parse(time.RFC3339, v); err != nil {
					invalidTypesPerColumn[colName] = append(invalidTypesPerColumn[colName], fmt.Sprintf("failed to parse datetime (value: %v): %v", v, err))
					row[colName] = nil
				} else {
					row[colName] = parsed
				}
			case nil:
				row[colName] = nil
			default:
				invalidTypesPerColumn[colName] = append(invalidTypesPerColumn[colName], fmt.Sprintf("%T", v))
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
					invalidTypesPerColumn[colName] = append(invalidTypesPerColumn[colName], fmt.Sprintf("failed to parse boolean (value: %v): %v", v, err))
					row[colName] = false
				} else {
					row[colName] = parsed
				}
			case int, int32, int64:
				row[colName] = v != 0
			case nil:
				row[colName] = false
			default:
				invalidTypesPerColumn[colName] = append(invalidTypesPerColumn[colName], fmt.Sprintf("%T", v))
				row[colName] = false
			}

		default:
			// do nothing
		}
	}

	return row, invalidTypesPerColumn
}
