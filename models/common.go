package models

import (
	"github.com/ohler55/ojg/alt"
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

// asMap marshals the given struct into a map.
func asMap(t any, removeFields ...string) map[string]any {
	v := alt.Decompose(t, &alt.Options{OmitEmpty: false, OmitNil: false})
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
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

// TODO: Find a better way to handle this
type ExtendedDBTable interface {
	DBTable
	Value() any
	PKCols() []clause.Column
}
