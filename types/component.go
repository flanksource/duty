package types

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/flanksource/commons/hash"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type ComponentStatus string

const (
	ComponentStatusHealthy   ComponentStatus = "healthy"
	ComponentStatusUnhealthy ComponentStatus = "unhealthy"
	ComponentStatusWarning   ComponentStatus = "warning"
	ComponentStatusError     ComponentStatus = "error"
	ComponentStatusInfo      ComponentStatus = "info"
)

var (
	ComponentStatusOrder = map[ComponentStatus]int{
		ComponentStatusInfo:      0,
		ComponentStatusHealthy:   1,
		ComponentStatusUnhealthy: 2,
		ComponentStatusWarning:   3,
		ComponentStatusError:     4,
	}
)

func (status ComponentStatus) Compare(other ComponentStatus) int {
	if status == other {
		return 0
	}
	if ComponentStatusOrder[status] > ComponentStatusOrder[other] {
		return 1
	}
	return -1
}

// +kubebuilder:object:generate=true
type Summary struct {
	Healthy   int                       `json:"healthy,omitempty"`
	Unhealthy int                       `json:"unhealthy,omitempty"`
	Warning   int                       `json:"warning,omitempty"`
	Info      int                       `json:"info,omitempty"`
	Incidents map[string]map[string]int `json:"incidents,omitempty"`
	Insights  map[string]map[string]int `json:"insights,omitempty"`
	Checks    map[string]int            `json:"checks,omitempty"`

	// processed is used to prevent from being caluclated twice
	processed bool `json:"-"`
}

func (s *Summary) SetProcessed(val bool) {
	s.processed = val
}

func (s Summary) IsProcessed() bool {
	return s.processed
}

func (s Summary) String() string {
	type _s Summary
	return fmt.Sprintf("%+v", _s(s))
}

func (s Summary) GetStatus() ComponentStatus {
	if s.Unhealthy > 0 {
		return ComponentStatusUnhealthy
	} else if s.Warning > 0 {
		return ComponentStatusWarning
	} else if s.Healthy > 0 {
		return ComponentStatusHealthy
	}
	return "unknown"
}

func (s Summary) Add(b Summary) Summary {
	if b.Healthy > 0 && b.Unhealthy > 0 {
		s.Warning += 1
	} else if b.Unhealthy > 0 {
		s.Unhealthy += 1
	} else if b.Healthy > 0 {
		s.Healthy += 1
	}
	if b.Warning > 0 {
		s.Warning += b.Warning
	}
	if b.Info > 0 {
		s.Info += b.Info
	}

	if s.Insights == nil {
		s.Insights = make(map[string]map[string]int)
	}
	for typ, details := range b.Insights {
		if _, exists := s.Insights[typ]; !exists {
			s.Insights[typ] = make(map[string]int)
		}
		for sev, count := range details {
			s.Insights[typ][sev] += count
		}
	}

	if s.Incidents == nil {
		s.Incidents = make(map[string]map[string]int)
	}
	for typ, details := range b.Incidents {
		if _, exists := s.Incidents[typ]; !exists {
			s.Incidents[typ] = make(map[string]int)
		}
		for sev, count := range details {
			s.Incidents[typ][sev] += count
		}
	}

	if s.Checks == nil {
		s.Checks = make(map[string]int)
	}
	for status, count := range b.Checks {
		s.Checks[status] += count
	}

	return s
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (s Summary) Value() (driver.Value, error) {
	return GenericStructValue(s, true)
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (s *Summary) Scan(val any) error {
	return GenericStructScan(&s, val)
}

// GormDataType gorm common data type
func (Summary) GormDataType() string {
	return "summary"
}

func (Summary) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (s Summary) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(s)
}

type ResourceSelectors []ResourceSelector

type ResourceSelector struct {
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`
}

func (rs *ResourceSelectors) Scan(val any) error {
	return GenericStructScan(&rs, val)
}

func (rs ResourceSelectors) Value() (driver.Value, error) {
	return GenericStructValue(rs, true)
}

func (rs ResourceSelectors) Hash() string {
	hash, err := hash.JSONMD5Hash(rs)
	if err != nil {
		return ""
	}
	return hash
}

// GormDataType gorm common data type
func (rs ResourceSelectors) GormDataType() string {
	return "resourceSelectors"
}

// GormDBDataType gorm db data type
func (ResourceSelectors) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (rs ResourceSelectors) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(rs)
}

type ComponentCheck struct {
	Selector ResourceSelector `json:"selector,omitempty"`
	Inline   *JSON            `json:"inline,omitempty"`
}

func (cs ComponentCheck) Hash() string {
	hash, err := hash.JSONMD5Hash(cs)
	if err != nil {
		return ""
	}
	return hash
}

type ComponentChecks []ComponentCheck

func (cs ComponentChecks) Value() (driver.Value, error) {
	return GenericStructValue(cs, true)
}

func (cs *ComponentChecks) Scan(val interface{}) error {
	return GenericStructScan(&cs, val)
}

// GormDataType gorm common data type
func (cs ComponentChecks) GormDataType() string {
	return "componentChecks"
}

// GormDBDataType gorm db data type
func (ComponentChecks) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return JSONGormDBDataType(db.Dialector.Name())
}

func (cs ComponentChecks) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return GormValue(cs)
}
