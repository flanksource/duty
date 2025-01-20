package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/flanksource/commons/hash"
	"github.com/google/uuid"
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

var ComponentStatusOrder = map[ComponentStatus]int{
	ComponentStatusInfo:      0,
	ComponentStatusHealthy:   1,
	ComponentStatusUnhealthy: 2,
	ComponentStatusWarning:   3,
	ComponentStatusError:     4,
}

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

func (t *Summary) AsEnv() map[string]any {
	return map[string]any{
		"healthy":   t.Healthy,
		"unhealthy": t.Unhealthy,
		"warning":   t.Warning,
		"info":      t.Info,
		"incidents": t.Incidents,
		"insights":  t.Insights,
		"checks":    t.Checks,
	}
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

// +kubebuilder:object:generate=true
type ComponentCheck struct {
	Selector ResourceSelector `json:"selector,omitempty"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Inline json.RawMessage `json:"inline" gorm:"type:JSON"`
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

type Incident struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Severity    int       `json:"severity"`
	Description string    `json:"description"`
}
