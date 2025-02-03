package models

import (
	"encoding/json"

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
		if h == HealthUnhealthy {
			return HealthUnhealthy
		} else if h == HealthWarning {
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
	NamespaceScope() string
}

// genericTagsMatcher implements TagsMatchable
type genericTagsMatcher struct {
	Tags map[string]string
}

func (c genericTagsMatcher) Get(key string) string {
	return c.Tags[key]
}

func (c genericTagsMatcher) Has(key string) bool {
	_, ok := c.Tags[key]
	return ok
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

type genericFieldMatcher struct {
	Fields map[string]any
}

func (c genericFieldMatcher) Get(key string) string {
	val := c.Fields[key]
	switch v := val.(type) {
	case string:
		return v
	default:
		marshalled, _ := json.Marshal(v)
		return string(marshalled)
	}
}

func (c genericFieldMatcher) Has(key string) bool {
	_, ok := c.Fields[key]
	return ok
}

// ABACAttribute is the object passed to casbin for authorization checks.
//
// NOTE: the fields are not a pointer to avoid nil pointer checks in the casbin policy.
type ABACAttribute struct {
	Playbook  Playbook
	Component Component
	Config    ConfigItem
	Check     Check
}

type TaggableModel interface {
	GetTags() map[string]string
}

type LabelableModel interface {
	GetLabels() map[string]string
}
