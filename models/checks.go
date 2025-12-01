package models

import (
	"fmt"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/flanksource/duty/types"
)

type CheckHealthStatus string

const (
	CheckStatusHealthy   = "healthy"
	CheckStatusUnhealthy = "unhealthy"
)

func (c CheckHealthStatus) Pretty() api.Text {
	switch c {
	case CheckStatusHealthy:
		return clicky.Text("✓ ", "text-green-600").Append(string(c), "capitalize text-green-600")
	case CheckStatusUnhealthy:
		return clicky.Text("✗ ", "text-red-600").Append(string(c), "capitalize text-red-600")
	default:
		return clicky.Text(string(c), "text-gray-500")
	}
}

var CheckHealthStatuses = []CheckHealthStatus{
	CheckStatusHealthy,
	CheckStatusUnhealthy,
}

// Ensure interface compliance
var (
	_ types.ResourceSelectable = Check{}
	_ LabelableModel           = Check{}
)

type Check struct {
	ID                 uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	CanaryID           uuid.UUID           `json:"canary_id"`
	AgentID            uuid.UUID           `json:"agent_id,omitempty"`
	Spec               types.JSON          `json:"-"`
	Type               string              `json:"type"`
	Name               string              `json:"name"`
	Namespace          string              `json:"namespace"`
	Labels             types.JSONStringMap `json:"labels" gorm:"type:jsonstringmap"`
	Description        string              `json:"description,omitempty"`
	Status             CheckHealthStatus   `json:"status,omitempty"`
	Owner              string              `json:"owner,omitempty"`
	Severity           Severity            `json:"severity,omitempty"`
	Icon               string              `json:"icon,omitempty"`
	Transformed        bool                `json:"transformed,omitempty"`
	LastTransitionTime *time.Time          `json:"last_transition_time,omitempty"`
	CreatedAt          *time.Time          `json:"created_at,omitempty" gorm:"<-:create"`
	UpdatedAt          *time.Time          `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt          *time.Time          `json:"deleted_at,omitempty"`
	SilencedAt         *time.Time          `json:"silenced_at,omitempty"`

	// Auxiliary fields
	CanaryName   string        `json:"canary_name,omitempty" gorm:"-"`
	ComponentIDs []string      `json:"components,omitempty"  gorm:"-"` // Linked component ids
	Uptime       types.Uptime  `json:"uptime,omitempty"  gorm:"-"`
	Latency      types.Latency `json:"latency,omitempty"  gorm:"-"`
	Statuses     []CheckStatus `json:"checkStatuses,omitempty"  gorm:"-"`
	DisplayType  string        `json:"display_type,omitempty"  gorm:"-"`

	// These are calculated for the selected date range
	EarliestRuntime *time.Time `json:"earliestRuntime,omitempty" gorm:"-"`
	LatestRuntime   *time.Time `json:"latestRuntime,omitempty" gorm:"-"`
	TotalRuns       int        `json:"totalRuns,omitempty" gorm:"-"`
}

func (c Check) Pretty() api.Text {
	t := c.Status.Pretty().AddText(" ")
	t = t.AddText(c.Name, "font-bold")

	if c.Type != "" {
		t = t.AddText(" ").Add(clicky.Text(c.Type, "text-xs text-cyan-600 bg-cyan-50"))
	}

	if c.Severity != "" {
		t = t.AddText(" ")
		t = t.Add(c.Severity.Pretty())
	}

	if c.Namespace != "" {
		t = t.AddText(" 📦 ", "text-gray-500").AddText(c.Namespace, "text-sm text-gray-600")
	}

	if c.Description != "" {
		t = t.NewLine().AddText("  "+c.Description, "text-sm text-gray-600")
	}

	return t
}

func (c Check) PrettyRow(opts interface{}) map[string]api.Text {
	row := map[string]api.Text{
		"name":   clicky.Text(c.Name, "font-bold"),
		"type":   clicky.Text(c.Type, "text-cyan-600"),
		"status": c.Status.Pretty(),
	}

	if c.Severity != "" {
		row["severity"] = c.Severity.Pretty()
	}

	if c.Namespace != "" {
		row["namespace"] = clicky.Text(c.Namespace, "text-blue-600")
	}

	// if c.LastRuntime != nil {
	// 	row["last_run"] = api.Human(time.Since(*c.LastRuntime), "text-gray-600")
	// }

	if c.CreatedAt != nil {
		row["age"] = api.Human(time.Since(*c.CreatedAt), "text-gray-600")
	}

	return row
}

func (t Check) Value() any {
	return &t
}

func (t Check) PKCols() []clause.Column {
	return []clause.Column{{Name: "id"}}
}

func (t Check) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(Check).CanaryID.String()
	})

	return db.Model(&Canary{}).Where("id IN ?", parentIDs).Update("is_pushed", false).Error
}

func (t Check) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Check
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Check, _ int) DBTable { return i }), err
}

func (c Check) PK() string {
	return c.ID.String()
}

func (c Check) TableName() string {
	return "checks"
}

func (t Check) GetLabels() map[string]string {
	return t.Labels
}

func (t Check) GetTrimmedLabels() []Label {
	return sortedTrimmedLabels(defaultLabelsWhitelist, defaultLabelsOrder, nil, t.Labels)
}

func (c Check) ToString() string {
	return fmt.Sprintf("%s-%s-%s", c.Name, c.Type, c.Description)
}

func (c Check) GetDescription() string {
	return c.Description
}

func (c Check) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}

func (c Check) GetID() string {
	return c.ID.String()
}

func (c Check) GetName() string {
	return c.Name
}

func (c Check) GetNamespace() string {
	return c.Namespace
}

func (c Check) GetType() string {
	return c.Type
}

func (c Check) GetAgentID() string {
	if c.AgentID == uuid.Nil {
		return ""
	}
	return c.AgentID.String()
}

func (c Check) GetStatus() (string, error) {
	return string(c.Status), nil
}

func (c Check) GetHealthDescription() string {
	return c.Description
}

func (c Check) GetHealth() (string, error) {
	if c.Status == CheckStatusHealthy {
		return string(HealthHealthy), nil
	}

	return string(HealthUnhealthy), nil
}

func (c Check) GetLabelsMatcher() labels.Labels {
	return checkLabelsProvider{c}
}

func (c Check) GetFieldsMatcher() fields.Fields {
	return types.GenericFieldMatcher{Fields: c.AsMap()}
}

type checkLabelsProvider struct {
	Check
}

func (c checkLabelsProvider) Get(key string) string {
	return c.Labels[key]
}

func (c checkLabelsProvider) Has(key string) bool {
	_, ok := c.Labels[key]
	return ok
}

func (c checkLabelsProvider) Lookup(key string) (string, bool) {
	value, ok := c.Labels[key]
	return value, ok
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

type ChecksUnlogged struct {
	CheckID     uuid.UUID  `json:"check_id" gorm:"primaryKey"`
	CanaryID    uuid.UUID  `json:"canary_id"`
	Status      string     `json:"status"`
	LastRuntime *time.Time `json:"last_runtime,omitempty"`
	NextRuntime *time.Time `json:"next_runtime,omitempty"`
}

func (ChecksUnlogged) TableName() string {
	return "checks_unlogged"
}

func (ChecksUnlogged) PK() string {
	return "check_id"
}

func (ChecksUnlogged) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ChecksUnlogged
	err := db.Select("checks_unlogged.*").
		Joins("LEFT JOIN checks ON checks_unlogged.check_id = checks.id").
		Where("checks.agent_id = ?", uuid.Nil).
		Where("checks_unlogged.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i ChecksUnlogged, _ int) DBTable { return i }), err
}

func (ChecksUnlogged) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(ChecksUnlogged)
		return []string{c.CheckID.String()}
	})

	return db.Model(&ChecksUnlogged{}).Where("check_id IN ?", ids).Update("is_pushed", true).Error
}

func (ChecksUnlogged) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(ChecksUnlogged).CheckID.String()
	})

	return db.Model(&Check{}).Where("id IN ?", parentIDs).Update("is_pushed", false).Error
}

type CheckStatus struct {
	CheckID   uuid.UUID `json:"check_id" gorm:"primaryKey"`
	Status    bool      `json:"status"`
	Invalid   bool      `json:"invalid,omitempty"`
	Time      string    `json:"time" gorm:"primaryKey"`
	Duration  int       `json:"duration"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	Detail    any       `json:"-" gorm:"-"`
	CreatedAt time.Time `json:"created_at,omitempty" gorm:"<-:create"`
	// IsPushed when set to true indicates that the check status has been pushed to upstream.
	IsPushed bool `json:"is_pushed,omitempty"`
}

func (c CheckStatus) Pretty() api.Text {
	var icon, style string
	if c.Status {
		icon, style = "✓", "text-green-600"
	} else {
		icon, style = "✗", "text-red-600"
	}

	t := clicky.Text(icon+" ", style)

	if c.Invalid {
		t = t.AddText("Invalid", "font-bold text-orange-600")
	} else if c.Status {
		t = t.AddText("Passed", "font-bold text-green-600")
	} else {
		t = t.AddText("Failed", "font-bold text-red-600")
	}

	if c.Duration > 0 {
		duration := time.Duration(c.Duration) * time.Millisecond
		t = t.AddText(" • ", "text-gray-400")
		t = t.Add(api.Human(duration, "text-gray-600"))
	}

	if c.Message != "" {
		t = t.NewLine().AddText("  "+c.Message, "text-sm text-gray-600")
	}

	if c.Error != "" {
		t = t.NewLine().AddText("  "+c.Error, "text-sm text-red-600")
	}

	return t
}

func (c CheckStatus) PrettyRow(opts interface{}) map[string]api.Text {
	row := map[string]api.Text{
		"time": clicky.Text(c.Time, "font-mono text-xs text-gray-600"),
	}

	if c.Status {
		row["status"] = clicky.Text("✓ Passed", "text-green-600")
	} else {
		row["status"] = clicky.Text("✗ Failed", "text-red-600")
	}

	if c.Invalid {
		row["status"] = clicky.Text("! Invalid", "text-orange-600")
	}

	if c.Duration > 0 {
		duration := time.Duration(c.Duration) * time.Millisecond
		row["duration"] = api.Human(duration, "text-gray-600")
	}

	if c.Message != "" {
		row["message"] = clicky.Text(c.Message, "text-gray-700")
	}

	if c.Error != "" {
		row["error"] = clicky.Text(c.Error, "text-red-600 text-sm")
	}

	return row
}

func (t CheckStatus) Value() any {
	return &t
}

func (t CheckStatus) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(CheckStatus).CheckID.String()
	})

	return db.Model(&Check{}).Where("id IN ?", parentIDs).Update("is_pushed", false).Error
}

func (s CheckStatus) PKCols() []clause.Column {
	return []clause.Column{{Name: "check_id"}, {Name: "time"}}
}

func (s CheckStatus) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []any {
		c := any(a).(CheckStatus)
		return []any{c.CheckID, c.Time}
	})

	return db.Model(&CheckStatus{}).Where("(check_id, time) IN ?", ids).Update("is_pushed", true).Error
}

func (s CheckStatus) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []CheckStatus
	err := db.Select("check_statuses.*").
		Joins("LEFT JOIN checks ON checks.id = check_statuses.check_id").
		Where("checks.agent_id = ?", uuid.Nil).
		Where("check_statuses.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i CheckStatus, _ int) DBTable { return i }), err
}

func (s CheckStatus) PK() string {
	return s.CheckID.String() + s.Time
}

func (s CheckStatus) GetTime() (time.Time, error) {
	return time.Parse(time.DateTime, s.Time)
}

func (CheckStatus) TableName() string {
	return "check_statuses"
}

func (s CheckStatus) AsMap(removeFields ...string) map[string]any {
	return asMap(s, removeFields...)
}

// CheckStatusAggregate1h represents the `check_statuses_1h` table
type CheckStatusAggregate1h struct {
	CheckID   string    `gorm:"column:check_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	Duration  int       `gorm:"column:duration"`
	Total     int       `gorm:"column:total"`
	Passed    int       `gorm:"column:passed"`
	Failed    int       `gorm:"column:failed"`
}

func (CheckStatusAggregate1h) TableName() string {
	return "check_statuses_1h"
}

// CheckStatusAggregate1d represents the `check_statuses_1d` table
type CheckStatusAggregate1d struct {
	CheckID   string    `gorm:"column:check_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	Duration  int       `gorm:"column:duration"`
	Total     int       `gorm:"column:total"`
	Passed    int       `gorm:"column:passed"`
	Failed    int       `gorm:"column:failed"`
}

func (CheckStatusAggregate1d) TableName() string {
	return "check_statuses_1d"
}

// CheckSummary represents the `check_summary` view
type CheckSummary struct {
	ID                 uuid.UUID           `json:"id"`
	CanaryID           uuid.UUID           `json:"canary_id"`
	CanaryName         string              `json:"canary_name"`
	CanaryNamespace    string              `json:"canary_namespace"`
	Description        string              `json:"description,omitempty"`
	Icon               string              `json:"icon,omitempty"`
	Labels             types.JSONStringMap `json:"labels"`
	LastTransitionTime *time.Time          `json:"last_transition_time,omitempty"`
	Latency            types.Latency       `json:"latency,omitempty"`
	Name               string              `json:"name"`
	Namespace          string              `json:"namespace"`
	Owner              string              `json:"owner,omitempty"`
	Severity           string              `json:"severity,omitempty"`
	Status             string              `json:"status"`
	Type               string              `json:"type"`
	Uptime             types.Uptime        `json:"uptime,omitempty"`
	LastRuntime        *time.Time          `json:"last_runtime,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	DeletedAt          *time.Time          `json:"deleted_at,omitempty"`
	SilencedAt         *time.Time          `json:"silenced_at,omitempty"`
}

func (t *CheckSummary) TableName() string {
	return "check_summary"
}

func (t CheckSummary) AsMap(removeFields ...string) map[string]any {
	return asMap(t, removeFields...)
}

type CheckConfigRelationship struct {
	ConfigID   uuid.UUID  `json:"config_id,omitempty"`
	CheckID    uuid.UUID  `json:"check_id,omitempty"`
	CanaryID   uuid.UUID  `json:"canary_id,omitempty"`
	SelectorID string     `json:"selector_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func (s CheckConfigRelationship) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(CheckConfigRelationship)
		return []string{c.ConfigID.String(), c.CheckID.String(), c.CanaryID.String(), c.SelectorID}
	})

	return db.Model(&CheckConfigRelationship{}).Where("(config_id, check_id, canary_id, selector_id) IN ?", ids).Update("is_pushed", true).Error
}

func (c CheckConfigRelationship) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []CheckConfigRelationship
	err := db.Select("check_config_relationships.*").
		Joins("LEFT JOIN config_items ci ON check_config_relationships.config_id = ci.id").
		Where("ci.agent_id = ?", uuid.Nil).
		Where("check_config_relationships.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i CheckConfigRelationship, _ int) DBTable { return i }), err
}

func (c *CheckConfigRelationship) Save(db *gorm.DB) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "canary_id"}, {Name: "check_id"}, {Name: "config_id"}, {Name: "selector_id"}},
		UpdateAll: true,
	}).Create(c).Error
}

func (c CheckConfigRelationship) PK() string {
	return c.ConfigID.String() + "," + c.CheckID.String() + "," + c.CanaryID.String() + "," + c.SelectorID
}

func (CheckConfigRelationship) TableName() string {
	return "check_config_relationships"
}
