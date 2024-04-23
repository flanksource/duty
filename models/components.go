package models

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

var AllowedColumnFieldsInComponents = []string{
	"owner",
	"topology_type",
	"topology_id",
	"parent_id",
	"type", // Deprecated. Use resource_selector.types instead
}

type Component struct {
	ID              uuid.UUID               `json:"id,omitempty" gorm:"default:generate_ulid()"` //nolint
	TopologyID      *uuid.UUID              `json:"topology_id,omitempty"`
	AgentID         uuid.UUID               `json:"agent_id,omitempty"`
	ExternalId      string                  `json:"external_id,omitempty"` //nolint
	ParentId        *uuid.UUID              `json:"parent_id,omitempty"`   //nolint
	Name            string                  `json:"name,omitempty"`
	Text            string                  `json:"text,omitempty"`
	TopologyType    string                  `json:"topology_type,omitempty"`
	Namespace       string                  `json:"namespace,omitempty"`
	Labels          types.JSONStringMap     `json:"labels,omitempty" gorm:"default:null"`
	Hidden          bool                    `json:"hidden,omitempty"`
	Silenced        bool                    `json:"silenced,omitempty"`
	Status          types.ComponentStatus   `json:"status,omitempty"`
	Description     string                  `json:"description,omitempty"`
	Lifecycle       string                  `json:"lifecycle,omitempty"`
	LogSelectors    types.LogSelectors      `json:"logs,omitempty" gorm:"column:log_selectors;default:null"`
	Tooltip         string                  `json:"tooltip,omitempty"`
	StatusReason    string                  `json:"status_reason,omitempty"`
	Schedule        string                  `json:"schedule,omitempty"`
	Icon            string                  `json:"icon,omitempty"`
	Type            string                  `json:"type,omitempty"`
	Owner           string                  `json:"owner,omitempty"`
	Selectors       types.ResourceSelectors `json:"selectors,omitempty" gorm:"resourceSelectors;default:null" swaggerignore:"true"`
	Configs         types.ConfigQueries     `json:"configs,omitempty" gorm:"default:null"`
	ComponentChecks types.ComponentChecks   `json:"componentChecks,omitempty" gorm:"default:null"`
	Properties      Properties              `json:"properties,omitempty" gorm:"type:properties;default:null"`
	Path            string                  `json:"path,omitempty"`
	Summary         types.Summary           `json:"summary,omitempty" gorm:"type:summary;default:null"`
	IsLeaf          bool                    `json:"is_leaf"`
	CostPerMinute   float64                 `json:"cost_per_minute,omitempty" gorm:"column:cost_per_minute"`
	CostTotal1d     float64                 `json:"cost_total_1d,omitempty" gorm:"column:cost_total_1d"`
	CostTotal7d     float64                 `json:"cost_total_7d,omitempty" gorm:"column:cost_total_7d"`
	CostTotal30d    float64                 `json:"cost_total_30d,omitempty" gorm:"column:cost_total_30d"`
	CreatedBy       *uuid.UUID              `json:"created_by,omitempty"`
	CreatedAt       time.Time               `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP();<-:false"`
	UpdatedAt       *time.Time              `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"autoUpdateTime:false"`
	DeletedAt       *time.Time              `json:"deleted_at,omitempty" time_format:"postgres_timestamp" swaggerignore:"true"`
	// Auxiliary fields
	Checks         map[string]int            `json:"checks,omitempty" gorm:"-"`
	Incidents      map[string]map[string]int `json:"incidents,omitempty" gorm:"-"`
	Analysis       map[string]map[string]int `json:"analysis,omitempty" gorm:"-"`
	Components     Components                `json:"components,omitempty" gorm:"-"`
	Order          int                       `json:"order,omitempty"  gorm:"-"`
	SelectorID     string                    `json:"-" gorm:"-"`
	RelationshipID *uuid.UUID                `json:"relationship_id,omitempty" gorm:"-"`
	Children       []string                  `json:"children,omitempty" gorm:"-"`
	Parents        []string                  `json:"parents,omitempty" gorm:"-"`

	// Mark it as true when the component is processed
	// during topology tree creation
	NodeProcessed bool `json:"-" gorm:"-"`
}

func (t Component) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Component
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Component, _ int) DBTable { return i }), err
}

func (c Component) PK() string {
	return c.ID.String()
}

func (c Component) TableName() string {
	return "components"
}

func (c *Component) ObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      c.Name,
		Namespace: c.Namespace,
		Labels:    c.Labels,
	}
}

func (c Component) GetStatus() string {
	if c.Summary.Healthy > 0 && c.Summary.Unhealthy > 0 {
		return string(types.ComponentStatusWarning)
	} else if c.Summary.Unhealthy > 0 {
		return string(types.ComponentStatusUnhealthy)
	} else if c.Summary.Warning > 0 {
		return string(types.ComponentStatusWarning)
	} else if c.Summary.Healthy > 0 {
		return string(types.ComponentStatusHealthy)
	} else {
		return string(types.ComponentStatusInfo)
	}
}

func (c *Component) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}

func (component Component) GetAsEnvironment() map[string]interface{} {
	return map[string]interface{}{
		"self":       component,
		"properties": component.Properties.AsMap(),
	}
}

func (c *Component) Summarize(depth int) types.Summary {
	if depth <= 0 {
		return c.Summary
	}
	if c.Summary.IsProcessed() {
		return c.Summary
	}

	var s types.Summary
	s.Incidents = c.Summary.Incidents
	s.Insights = c.Summary.Insights
	s.Checks = c.Summary.Checks

	if c.Components == nil {
		switch types.ComponentStatus(c.Status) {
		case types.ComponentStatusHealthy:
			s.Healthy++
		case types.ComponentStatusUnhealthy:
			s.Unhealthy++
		case types.ComponentStatusWarning:
			s.Warning++
		case types.ComponentStatusInfo:
			s.Info++
		}

		s.SetProcessed(true)
		return s
	}

	for _, child := range c.Components {
		childSummary := child.Summarize(depth - 1)
		s = s.Add(childSummary)
	}
	s.SetProcessed(true)
	return s
}

func (component Component) Clone() Component {
	clone := Component{
		Name:         component.Name,
		TopologyType: component.TopologyType,
		Order:        component.Order,
		ID:           component.ID,
		Text:         component.Text,
		Namespace:    component.Namespace,
		Labels:       component.Labels,
		Tooltip:      component.Tooltip,
		Icon:         component.Icon,
		Owner:        component.Owner,
		Status:       component.Status,
		StatusReason: component.StatusReason,
		Type:         component.Type,
		Lifecycle:    component.Lifecycle,
		Checks:       component.Checks,
		Configs:      component.Configs,
		Properties:   component.Properties,
		ExternalId:   component.ExternalId,
		Schedule:     component.Schedule,
	}

	copy(clone.LogSelectors, component.LogSelectors)
	return clone
}

func (component Component) String() string {
	s := ""
	if component.Type != "" {
		s += component.Type + "/"
	}
	if component.Namespace != "" {
		s += component.Namespace + "/"
	}
	if component.Text != "" {
		s += component.Text
	} else if component.Name != "" {
		s += component.Name
	} else {
		s += component.ExternalId
	}
	return s
}

func (component Component) IsHealthy() bool {
	s := component.Summarize(10)
	return s.Healthy > 0 && s.Unhealthy == 0 && s.Warning == 0
}

func (c Component) GetID() string {
	return c.ID.String()
}

func (c Component) GetName() string {
	return c.Name
}

func (c Component) GetNamespace() string {
	return c.Namespace
}

func (c Component) GetType() string {
	return c.Type
}

func (c Component) GetLabelsMatcher() labels.Labels {
	return componentLabelsProvider{c}
}

func (c Component) GetFieldsMatcher() fields.Fields {
	return componentFieldsProvider{c}
}

type componentLabelsProvider struct {
	Component
}

func (c componentLabelsProvider) Get(key string) string {
	return c.Labels[key]
}

func (c componentLabelsProvider) Has(key string) bool {
	_, ok := c.Labels[key]
	return ok
}

type componentFieldsProvider struct {
	Component
}

func (c componentFieldsProvider) Get(key string) string {
	if lo.Contains(AllowedColumnFieldsInComponents, key) {
		return fmt.Sprintf("%v", c.AsMap()[key])
	}

	v := c.Properties.Find(key)
	if v == nil {
		return ""
	}

	return fmt.Sprintf("%v", v.GetValue())
}

func (c componentFieldsProvider) Has(key string) bool {
	if lo.Contains(AllowedColumnFieldsInComponents, key) {
		_, ok := c.AsMap()[key]
		return ok
	}

	v := c.Properties.Find(key)
	return v != nil
}

var ComponentID = func(c Component) string {
	return c.ID.String()
}

var CheckID = func(c Check) string {
	return c.ID.String()
}

type Components []*Component

func (components Components) Map(fn func(c *Component)) {
	for _, c := range components {
		fn(c)
		if c.Components != nil {
			c.Components.Map(fn)
		}
	}
}

func (components Components) Debug(prefix string) string {
	var s string
	for _, component := range components {
		status := string(component.Status)

		if component.IsHealthy() {
			status = console.Greenf("%s", status)
		} else {
			status = console.Redf("%s", status)
		}

		s += fmt.Sprintf("%s%s (id=%s, text=%s, name=%s) => %s\n", prefix, component, component.ID, component.Text, component.Name, status)
		s += component.Components.Debug(prefix + "\t")
	}

	return s
}

func (components Components) Summarize(depth int) types.Summary {
	var s types.Summary
	for _, component := range components {
		s = s.Add(component.Summarize(depth))
	}

	return s
}

func (components Components) Walk() Components {
	var comps Components
	for _, _c := range components {
		c := _c
		comps = append(comps, c)
		if c.Components != nil {
			comps = append(comps, c.Components.Walk()...)
		}
	}

	return comps
}

func (components Components) Find(name string) *Component {
	for _, component := range components {
		if component.Name == name {
			return component
		}
	}
	return nil
}

type Text struct {
	Tooltip string `json:"tooltip,omitempty"`
	Icon    string `json:"icon,omitempty"`
	Text    string `json:"text,omitempty"`
	Label   string `json:"label,omitempty"`
}

// +kubebuilder:object:generate=true
type Link struct {
	// e.g. documentation, support, playbook
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
	Text `json:",inline"`
}

// Property is a realized v1.Property without the lookup definition
// +kubebuilder:object:generate=true
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

func (p Property) GetValue() any {
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

func (p Property) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
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

func (p Properties) AsMap() map[string]any {
	result := make(map[string]any)
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
	return types.GenericStructValue(p, true)
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (p *Properties) Scan(val any) error {
	return types.GenericStructScan(&p, val)
}

// GormDataType gorm common data type
func (Properties) GormDataType() string {
	return "properties"
}

func (Properties) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return types.JSONGormDBDataType(db.Dialector.Name())
}

func (p Properties) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return types.GormValue(p)
}

type ComponentRelationship struct {
	ComponentID      uuid.UUID  `json:"component_id,omitempty"`
	RelationshipID   uuid.UUID  `json:"relationship_id,omitempty"`
	SelectorID       string     `json:"selector_id,omitempty"`
	RelationshipPath string     `json:"relationship_path,omitempty"`
	CreatedAt        time.Time  `json:"created_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

func (s ComponentRelationship) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(ComponentRelationship)
		return []string{c.ComponentID.String(), c.RelationshipID.String(), c.SelectorID}
	})

	return db.Model(&ComponentRelationship{}).Where("(component_id, relationship_id, selector_id) IN ?", ids).Update("is_pushed", true).Error
}

func (cr ComponentRelationship) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ComponentRelationship
	err := db.Select("component_relationships.*").
		Joins("LEFT JOIN components c ON component_relationships.component_id = c.id").
		Joins("LEFT JOIN components rel ON component_relationships.relationship_id = rel.id").
		Where("c.agent_id = ? AND rel.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("component_relationships.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i ComponentRelationship, _ int) DBTable { return i }), err
}

func (cr ComponentRelationship) PK() string {
	return cr.ComponentID.String() + "," + cr.RelationshipID.String() + "," + cr.SelectorID
}

func (ComponentRelationship) TableName() string {
	return "component_relationships"
}

type ConfigComponentRelationship struct {
	ComponentID uuid.UUID  `json:"component_id,omitempty"`
	ConfigID    uuid.UUID  `json:"config_id,omitempty"`
	SelectorID  string     `json:"selector_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (s ConfigComponentRelationship) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(ConfigComponentRelationship)
		return []string{c.ComponentID.String(), c.ConfigID.String()}
	})

	return db.Model(&ConfigComponentRelationship{}).Where("(component_id, config_id) IN ?", ids).Update("is_pushed", true).Error
}

func (t ConfigComponentRelationship) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigComponentRelationship
	err := db.Select("config_component_relationships.*").
		Joins("LEFT JOIN components c ON config_component_relationships.component_id = c.id").
		Joins("LEFT JOIN config_items ci ON config_component_relationships.config_id = ci.id").
		Where("c.agent_id = ? AND ci.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("config_component_relationships.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i ConfigComponentRelationship, _ int) DBTable { return i }), err
}

func (t ConfigComponentRelationship) PK() string {
	return t.ComponentID.String() + "," + t.ConfigID.String()
}

func (ConfigComponentRelationship) TableName() string {
	return "config_component_relationships"
}

var ConfigID = func(c ConfigComponentRelationship, i int) string {
	return c.ConfigID.String()
}

var ConfigSelectorID = func(c ConfigComponentRelationship, i int) string {
	return c.SelectorID
}

type CheckComponentRelationship struct {
	ComponentID uuid.UUID  `json:"component_id,omitempty"`
	CheckID     uuid.UUID  `json:"check_id,omitempty"`
	CanaryID    uuid.UUID  `json:"canary_id,omitempty"`
	SelectorID  string     `json:"selector_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (s CheckComponentRelationship) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(CheckComponentRelationship)
		return []string{c.ComponentID.String(), c.CheckID.String(), c.CanaryID.String(), c.SelectorID}
	})

	return db.Model(&CheckComponentRelationship{}).Where("(component_id, check_id, canary_id, selector_id) IN ?", ids).Update("is_pushed", true).Error
}

func (t CheckComponentRelationship) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []CheckComponentRelationship
	err := db.Select("check_component_relationships.*").
		Joins("LEFT JOIN components c ON check_component_relationships.component_id = c.id").
		Joins("LEFT JOIN canaries ON check_component_relationships.canary_id = canaries.id").
		Where("c.agent_id = ? AND canaries.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("check_component_relationships.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i CheckComponentRelationship, _ int) DBTable { return i }), err
}

func (c *CheckComponentRelationship) Save(db *gorm.DB) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "canary_id"}, {Name: "check_id"}, {Name: "component_id"}, {Name: "selector_id"}},
		UpdateAll: true,
	}).Create(c).Error
}

func (c CheckComponentRelationship) PK() string {
	return c.ComponentID.String() + "," + c.CheckID.String() + "," + c.CanaryID.String() + "," + c.SelectorID
}

func (CheckComponentRelationship) TableName() string {
	return "check_component_relationships"
}
