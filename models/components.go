package models

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type Components []*Component

type Component struct {
	Checks           Checks                   `json:"checks,omitempty" gorm:"-"`
	ComponentChecks  ComponentChecks          `json:"-" gorm:"column:component_checks" swaggerignore:"true"`
	Components       Components               `json:"components,omitempty" gorm:"-"`
	ConfigInsights   []map[string]interface{} `json:"insights,omitempty" gorm:"-"`
	Configs          Configs                  `json:"configs,omitempty" gorm:"type:configs"`
	CostPerMinute    float64                  `json:"cost_per_minute,omitempty" gorm:"column:cost_per_minute"`
	CostTotal1d      float64                  `json:"cost_total_1d,omitempty" gorm:"column:cost_total_1d"`
	CostTotal30d     float64                  `json:"cost_total_30d,omitempty" gorm:"column:cost_total_30d"`
	CostTotal7d      float64                  `json:"cost_total_7d,omitempty" gorm:"column:cost_total_7d"`
	CreatedAt        time.Time                `json:"created_at,omitempty" time_format:"postgres_timestamp"`
	DeletedAt        *time.Time               `json:"deleted_at,omitempty" time_format:"postgres_timestamp" swaggerignore:"true"`
	ExternalId       string                   `json:"external_id,omitempty"` //nolint
	Icon             string                   `json:"icon,omitempty"`
	ID               uuid.UUID                `json:"id,omitempty" gorm:"default:generate_ulid()"` //nolint
	Incidents        []Incident               `json:"incidents,omitempty" gorm:"-"`
	IsLeaf           bool                     `json:"is_leaf"`
	Labels           types.JSONStringMap      `json:"labels,omitempty"`
	Lifecycle        string                   `json:"lifecycle,omitempty"`
	Name             string                   `json:"name,omitempty"`
	Namespace        string                   `json:"namespace,omitempty"`
	Order            int                      `json:"order,omitempty"  gorm:"-"`
	Owner            string                   `json:"owner,omitempty"`
	ParentId         *uuid.UUID               `json:"parent_id,omitempty"` //nolint
	Path             string                   `json:"path,omitempty"`
	Properties       Properties               `json:"properties,omitempty" gorm:"type:properties"`
	Schedule         string                   `json:"schedule,omitempty"`
	SelectorID       string                   `json:"-" gorm:"-"`
	Selectors        ResourceSelectors        `json:"selectors,omitempty" gorm:"resourceSelectors" swaggerignore:"true"`
	Status           ComponentStatus          `json:"status,omitempty"`
	StatusReason     string                   `json:"statusReason,omitempty"`
	Summary          Summary                  `json:"summary,omitempty" gorm:"type:summary"`
	SystemTemplateID *uuid.UUID               `json:"system_template_id,omitempty"`
	Text             string                   `json:"text,omitempty"`
	Tooltip          string                   `json:"tooltip,omitempty"`
	TopologyType     string                   `json:"topology_type,omitempty"`
	Type             string                   `json:"type,omitempty"`
	UpdatedAt        time.Time                `json:"updated_at,omitempty" time_format:"postgres_timestamp"`
}

type ComponentStatus string

type Incident struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Severity    int       `json:"severity"`
	Description string    `json:"description"`
}

type Text struct {
	Tooltip string `json:"tooltip,omitempty"`
	Icon    string `json:"icon,omitempty"`
	Text    string `json:"text,omitempty"`
	Label   string `json:"label,omitempty"`
}

type Link struct {
	// e.g. documentation, support, playbook
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
	Text `json:",inline"`
}

type CheckStatus struct {
	Status   bool        `json:"status"`
	Invalid  bool        `json:"invalid,omitempty"`
	Time     string      `json:"time"`
	Duration int         `json:"duration"`
	Message  string      `json:"message,omitempty"`
	Error    string      `json:"error,omitempty"`
	Detail   interface{} `json:"-"`
}

func (s CheckStatus) GetTime() (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s.Time)
}

type Latency struct {
	Percentile99 float64 `json:"p99,omitempty" db:"p99"`
	Percentile97 float64 `json:"p97,omitempty" db:"p97"`
	Percentile95 float64 `json:"p95,omitempty" db:"p95"`
	Rolling1H    float64 `json:"rolling1h"`
}

type Summary struct {
	Healthy   int                       `json:"healthy,omitempty"`
	Unhealthy int                       `json:"unhealthy,omitempty"`
	Warning   int                       `json:"warning,omitempty"`
	Info      int                       `json:"info,omitempty"`
	Incidents map[string]map[string]int `json:"incidents,omitempty"`
	Insights  map[string]map[string]int `json:"insights,omitempty"`
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (s Summary) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (s *Summary) Scan(val interface{}) error {
	if val == nil {
		*s = Summary{}
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal properties value:", val))
	}
	err := json.Unmarshal(ba, s)
	return err
}

// GormDataType gorm common data type
func (Summary) GormDataType() string {
	return "summary"
}

func (Summary) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case types.SqliteType:
		return types.Text
	case types.PostgresType:
		return types.JSONBType
	case types.SQLServerType:
		return types.NVarcharType
	}
	return ""
}

func (s Summary) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(s)
	return gorm.Expr("?", data)
}

// Property is a realized v1.Property without the lookup definition
type Property struct {
	Label    string `json:"label,omitempty"`
	Name     string `json:"name,omitempty"`
	Tooltip  string `json:"tooltip,omitempty"`
	Icon     string `json:"icon,omitempty"`
	Type     string `json:"type,omitempty"`
	Color    string `json:"color,omitempty"`
	Order    int    `json:"order,omitempty"`
	Headline bool   `json:"headline,omitempty"`

	// Either text or value is required, but not both.
	Text  string `json:"text,omitempty"`
	Value int64  `json:"value,omitempty"`

	// e.g. milliseconds, bytes, millicores, epoch etc.
	Unit string `json:"unit,omitempty"`
	Max  *int64 `json:"max,omitempty"`
	Min  int64  `json:"min,omitempty"`

	Status         string `json:"status,omitempty"`
	LastTransition string `json:"lastTransition,omitempty"`
	Links          []Link `json:"links,omitempty"`
}

func (p Property) GetValue() interface{} {
	if p.Text != "" {
		return p.Text
	}
	if p.Value != 0 {
		return p.Value
	}
	return nil
}

func (p *Property) String() string {
	s := fmt.Sprintf("%s[", p.Name)
	if p.Text != "" {
		s += fmt.Sprintf("text=%s ", p.Text)
	}
	if p.Value != 0 {
		s += fmt.Sprintf("value=%d ", p.Value)
	}
	if p.Unit != "" {
		s += fmt.Sprintf("unit=%s ", p.Unit)
	}
	if p.Max != nil {
		s += fmt.Sprintf("max=%d ", *p.Max)
	}
	if p.Min != 0 {
		s += fmt.Sprintf("min=%d ", p.Min)
	}
	if p.Status != "" {
		s += fmt.Sprintf("status=%s ", p.Status)
	}
	if p.LastTransition != "" {
		s += fmt.Sprintf("lastTransition=%s ", p.LastTransition)
	}

	return strings.TrimRight(s, " ") + "]"
}

func (p *Property) Merge(other *Property) {
	if other.Text != "" {
		p.Text = other.Text
	}
	if other.Value != 0 {
		p.Value = other.Value
	}
	if other.Unit != "" {
		p.Unit = other.Unit
	}
	if other.Max != nil {
		p.Max = other.Max
	}
	if other.Min != 0 {
		p.Min = other.Min
	}
	if other.Order > 0 {
		p.Order = other.Order
	}
	if other.Status != "" {
		p.Status = other.Status
	}
	if other.LastTransition != "" {
		p.LastTransition = other.LastTransition
	}
	if other.Links != nil {
		p.Links = other.Links
	}
	if other.Type != "" {
		p.Type = other.Type
	}
	if other.Color != "" {
		p.Color = other.Color
	}
}

type Properties []*Property

func (p Properties) AsJSON() []byte {
	if len(p) == 0 {
		return []byte("[]")
	}
	data, err := json.Marshal(p)
	if err != nil {
		logger.Errorf("Error marshalling properties: %v", err)
	}
	return data
}

func (p Properties) AsMap() map[string]interface{} {
	result := make(map[string]interface{})
	for _, property := range p {
		result[property.Name] = property.GetValue()
	}
	return result
}

func (p Properties) Find(name string) *Property {
	for _, prop := range p {
		if prop.Name == name {
			return prop
		}
	}
	return nil
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (p Properties) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return p.AsJSON(), nil
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (p *Properties) Scan(val interface{}) error {
	if val == nil {
		*p = make(Properties, 0)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal properties value:", val))
	}
	err := json.Unmarshal(ba, p)
	return err
}

// GormDataType gorm common data type
func (Properties) GormDataType() string {
	return "properties"
}

func (Properties) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case types.SqliteType:
		return types.Text
	case types.PostgresType:
		return types.JSONBType
	case types.SQLServerType:
		return types.NVarcharType
	}
	return ""
}

func (p Properties) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(p)
	return gorm.Expr("?", data)
}

type Check struct {
	ID          uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	CanaryID    uuid.UUID           `json:"canary_id"`
	Spec        types.JSON          `json:"-"`
	Type        string              `json:"type"`
	Name        string              `json:"name"`
	CanaryName  string              `json:"canary_name" gorm:"-"`
	Namespace   string              `json:"namespace"  gorm:"-"`
	Labels      types.JSONStringMap `json:"labels" gorm:"type:jsonstringmap"`
	Description string              `json:"description,omitempty"`
	Status      string              `json:"status,omitempty"`
	Uptime      Uptime              `json:"uptime"  gorm:"-"`
	Latency     Latency             `json:"latency"  gorm:"-"`
	Statuses    []CheckStatus       `json:"checkStatuses"  gorm:"-"`
	Owner       string              `json:"owner,omitempty"`
	Severity    string              `json:"severity,omitempty"`
	Icon        string              `json:"icon,omitempty"`
	DisplayType string              `json:"displayType,omitempty"  gorm:"-"`
	LastRuntime *time.Time          `json:"lastRuntime,omitempty"`
	NextRuntime *time.Time          `json:"nextRuntime,omitempty"`
	UpdatedAt   *time.Time          `json:"updatedAt,omitempty"`
	CreatedAt   *time.Time          `json:"createdAt,omitempty"`
	DeletedAt   *time.Time          `json:"deletedAt,omitempty"`
	// Canary      *v1.Canary          `json:"-" gorm:"-"`
}

func (c Check) ToString() string {
	return fmt.Sprintf("%s-%s-%s", c.Name, c.Type, c.Description)
}

func (c Check) GetDescription() string {
	return c.Description
}

type Checks []*Check

func (c Checks) Len() int {
	return len(c)
}

func (c Checks) Less(i, j int) bool {
	return c[i].ToString() < c[j].ToString()
}

func (c Checks) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Checks) Find(key string) *Check {
	for _, check := range c {
		if check.Name == key {
			return check
		}
	}
	return nil
}

type Uptime struct {
	Passed int     `json:"passed"`
	Failed int     `json:"failed"`
	P100   float64 `json:"p100,omitempty"`
}

func (u Uptime) String() string {
	if u.Passed == 0 && u.Failed == 0 {
		return ""
	}
	if u.Passed == 0 {
		return fmt.Sprintf("0/%d 0%%", u.Failed)
	}
	percentage := 100.0 * (1 - (float64(u.Failed) / float64(u.Passed+u.Failed)))
	return fmt.Sprintf("%d/%d (%0.1f%%)", u.Passed, u.Passed+u.Failed, percentage)
}

type ResourceSelectors []ResourceSelector

type ResourceSelector struct {
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`
}

func (rs *ResourceSelectors) Scan(val interface{}) error {
	if val == nil {
		*rs = ResourceSelectors{}
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal ResourceSelectors value:", val))
	}
	return json.Unmarshal(ba, rs)
}

func (rs ResourceSelectors) Value() (driver.Value, error) {
	if len(rs) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(rs)
}

// GormDataType gorm common data type
func (rs ResourceSelectors) GormDataType() string {
	return "resourceSelectors"
}

// GormDBDataType gorm db data type
func (ResourceSelectors) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case types.SqliteType:
		return types.JSONType
	case types.PostgresType:
		return types.JSONBType
	case types.SQLServerType:
		return types.NVarcharType
	}
	return ""
}

func (rs ResourceSelectors) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(rs)
	return gorm.Expr("?", string(data))
}

type ComponentChecks []ComponentCheck

type ComponentCheck struct {
	Selector ResourceSelector `json:"selector,omitempty"`
	Inline   *CanarySpec      `json:"inline,omitempty"`
}

func (cs ComponentChecks) Value() (driver.Value, error) {
	if len(cs) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(cs)
}

func (cs *ComponentChecks) Scan(val interface{}) error {
	if val == nil {
		*cs = ComponentChecks{}
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal componentChecks value:", val))
	}
	return json.Unmarshal(ba, cs)
}

// GormDataType gorm common data type
func (cs ComponentChecks) GormDataType() string {
	return "componentChecks"
}

// GormDBDataType gorm db data type
func (ComponentChecks) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case types.SqliteType:
		return types.JSONType
	case types.PostgresType:
		return types.JSONBType
	case types.SQLServerType:
		return types.NVarcharType
	}
	return ""
}

func (cs ComponentChecks) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := json.Marshal(cs)
	return gorm.Expr("?", string(data))
}

type ComponentRelationship struct {
	ComponentID      uuid.UUID  `gorm:"column:component_id" json:"component_id,omitempty"`
	RelationshipID   uuid.UUID  `gorm:"column:relationship_id" json:"relationship_id,omitempty"`
	SelectorID       string     `gorm:"column:selector_id" json:"selector_id,omitempty"`
	RelationshipPath string     `gorm:"column:relationship_path" json:"relationship_path,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at,omitempty"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at,omitempty"`
	DeletedAt        *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (cr ComponentRelationship) TableName() string {
	return "component_relationships"
}

type ConfigComponentRelationship struct {
	ComponentID uuid.UUID  `gorm:"column:component_id" json:"component_id,omitempty"`
	ConfigID    uuid.UUID  `gorm:"column:config_id" json:"config_id,omitempty"`
	SelectorID  string     `gorm:"column:selector_id" json:"selector_id,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at,omitempty"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at,omitempty"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (cr ConfigComponentRelationship) TableName() string {
	return "config_component_relationships"
}
