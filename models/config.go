package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/hash"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// Config classes
const (
	ConfigClassCluster        = "Cluster"
	ConfigClassDatabase       = "Database"
	ConfigClassDeployment     = "Deployment"
	ConfigClassNamespace      = "Namespace"
	ConfigClassNode           = "Node"
	ConfigClassPod            = "Pod"
	ConfigClassVirtualMachine = "VirtualMachine"
)

// Config Statuses
const (
	ConfigStatusCordoned   = "Cordoned"
	ConfigStatusCordoning  = "Cordoning"
	ConfigStatusDeleted    = "Deleted"
	ConfigStatusDeleting   = "Deleting"
	ConfigStatusFailed     = "Failed"
	ConfigStatusPending    = "Pending"
	ConfigStatusRunning    = "Running"
	ConfigStatusStarting   = "Starting"
	ConfigStatusStopped    = "Stopped"
	ConfigStatusStopping   = "Stopping"
	ConfigStatusSucceeded  = "Succeeded"
	ConfigStatusUncordoned = "Uncordoned"
	ConfigStatusUnknown    = "Unknown"
)

// Config Analysis statuses
const (
	AnalysisStatusOpen     = "open"
	AnalysisStatusResolved = "resolved"
	AnalysisStatusSilenced = "silenced"
)

type AnalysisType string

const (
	AnalysisTypeAvailability   AnalysisType = "availability"
	AnalysisTypeCompliance     AnalysisType = "compliance"
	AnalysisTypeCost           AnalysisType = "cost"
	AnalysisTypeIntegration    AnalysisType = "integration"
	AnalysisTypeOther          AnalysisType = "other"
	AnalysisTypePerformance    AnalysisType = "performance"
	AnalysisTypeRecommendation AnalysisType = "recommendation"
	AnalysisTypeReliability    AnalysisType = "reliability"
	AnalysisTypeSecurity       AnalysisType = "security"
	AnalysisTypeTechDebt       AnalysisType = "technical_debt"
)

// ConfigItem represents the config item database table
type ConfigItem struct {
	ID              uuid.UUID            `json:"id" faker:"uuid_hyphenated"`
	ScraperID       *string              `json:"scraper_id,omitempty"`
	AgentID         uuid.UUID            `json:"agent_id,omitempty"`
	ConfigClass     string               `json:"config_class" faker:"oneof:File,EC2Instance,KubernetesPod" `
	ExternalID      pq.StringArray       `gorm:"type:[]text" json:"external_id,omitempty"`
	Type            *string              `json:"type,omitempty"`
	Status          *string              `json:"status,omitempty" gorm:"default:null"`
	Name            *string              `json:"name,omitempty" faker:"name"  `
	Namespace       *string              `json:"namespace,omitempty" faker:"oneof: default, demo, prod, staging" `
	Description     *string              `json:"description,omitempty"`
	Config          *string              `json:"config,omitempty"  `
	Source          *string              `json:"source,omitempty"  `
	ParentID        *uuid.UUID           `json:"parent_id,omitempty" faker:"-"`
	Path            string               `json:"path,omitempty" faker:"-"`
	CostPerMinute   float64              `gorm:"column:cost_per_minute;default:null" json:"cost_per_minute,omitempty"`
	CostTotal1d     float64              `gorm:"column:cost_total_1d;default:null" json:"cost_total_1d,omitempty"`
	CostTotal7d     float64              `gorm:"column:cost_total_7d;default:null" json:"cost_total_7d,omitempty"`
	CostTotal30d    float64              `gorm:"column:cost_total_30d;default:null" json:"cost_total_30d,omitempty"`
	Tags            *types.JSONStringMap `json:"tags,omitempty" faker:"tags"`
	Properties      *types.Properties    `json:"properties,omitempty"`
	LastScrapedTime *time.Time           `json:"last_scraped_time,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       *time.Time           `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt       *time.Time           `json:"deleted_at,omitempty"`
	DeleteReason    string               `json:"delete_reason,omitempty"`
}

func (t ConfigItem) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigItem
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i ConfigItem, _ int) DBTable { return i }), err
}

func (t ConfigItem) PK() string {
	return t.ID.String()
}

func (ConfigItem) TableName() string {
	return "config_items"
}

func (ci *ConfigItem) SetParent(parent *ConfigItem) {
	ci.ParentID = &parent.ID
	ci.Path = parent.Path + "." + ci.ID.String()
}

func (ci ConfigItem) String() string {
	return fmt.Sprintf("%s{name=%s, id=%s}", ci.ConfigClass, *ci.Name, ci.ID)
}

func (ci ConfigItem) AsMap(removeFields ...string) map[string]any {
	return asMap(ci, removeFields...)
}

func (ci ConfigItem) ConfigJSONStringMap() (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal([]byte(*ci.Config), &m)
	return m, err
}

func (ci ConfigItem) TemplateEnv() (map[string]any, error) {
	env := ci.AsMap()
	if ci.Config == nil {
		return env, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(*ci.Config), &m); err != nil {
		return env, err
	}
	env["config"] = m
	return env, nil
}

func (c ConfigItem) GetSelectorID() string {
	if c.Config == nil || *c.Config == "" {
		return ""
	}

	selectorID, err := hash.JSONMD5Hash(c.Config)
	if err != nil {
		return ""
	}

	return selectorID
}

func (c ConfigItem) GetID() string {
	return c.ID.String()
}

func (c ConfigItem) GetName() string {
	if c.Name == nil {
		return ""
	}
	return *c.Name
}

func (c ConfigItem) GetNamespace() string {
	if c.Namespace == nil {
		return ""
	}
	return *c.Namespace
}

func (c ConfigItem) GetType() string {
	if c.Type == nil {
		return ""
	}
	return *c.Type
}

func (c ConfigItem) GetStatus() string {
	if c.Status == nil {
		return ""
	}
	return *c.Status
}

func (c ConfigItem) GetLabelsMatcher() labels.Labels {
	return configLabels{c}
}

func (c ConfigItem) GetFieldsMatcher() fields.Fields {
	return configFields{c}
}

type configFields struct {
	ConfigItem
}

var AllowedColumnFieldsInConfigs = []string{"config_class", "external_id"}

func (c configFields) Get(key string) string {
	if lo.Contains(AllowedColumnFieldsInConfigs, key) {
		return fmt.Sprintf("%v", c.AsMap()[key])
	}

	v := c.Properties.Find(key)
	if v == nil {
		return ""
	}

	return fmt.Sprintf("%v", v.GetValue())
}

func (c configFields) Has(key string) bool {
	if lo.Contains(AllowedColumnFieldsInConfigs, key) {
		_, ok := c.AsMap()[key]
		return ok
	}

	v := c.Properties.Find(key)
	return v != nil
}

type configLabels struct {
	ConfigItem
}

func (c configLabels) Get(key string) string {
	if c.Tags == nil || len(*c.Tags) == 0 {
		return ""
	}

	return (*c.Tags)[key]
}

func (c configLabels) Has(key string) bool {
	if c.Tags == nil || len(*c.Tags) == 0 {
		return false
	}

	_, ok := (*c.Tags)[key]
	return ok
}

// ConfigScraper represents the config_scrapers database table
type ConfigScraper struct {
	ID          uuid.UUID  `json:"id"`
	AgentID     uuid.UUID  `json:"agent_id,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Spec        string     `json:"spec,omitempty"`
	Source      string     `json:"source,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (t ConfigScraper) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigScraper
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i ConfigScraper, _ int) DBTable { return i }), err
}

func (c ConfigScraper) PK() string {
	return c.ID.String()
}

func (c ConfigScraper) TableName() string {
	return "config_scrapers"
}

func (c ConfigScraper) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}

// BeforeCreate GORM hook
func (cs *ConfigScraper) BeforeCreate(tx *gorm.DB) error {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}
	return nil
}

type ConfigRelationship struct {
	ConfigID   string     `json:"config_id"`
	RelatedID  string     `json:"related_id"`
	Relation   string     `json:"relation"`
	SelectorID string     `json:"selector_id"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func (s ConfigRelationship) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(ConfigRelationship)
		return []string{c.RelatedID, c.ConfigID, c.SelectorID}
	})

	return db.Model(&ConfigRelationship{}).Where("(related_id, config_id, selector_id) IN ?", ids).Update("is_pushed", true).Error
}

func (t ConfigRelationship) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigRelationship
	err := db.Select("config_relationships.*").
		Joins("LEFT JOIN config_items ci ON config_relationships.config_id = ci.id").
		Where("ci.agent_id = ?", uuid.Nil).
		Where("config_relationships.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i ConfigRelationship, _ int) DBTable { return i }), err
}

func (cr ConfigRelationship) PK() string {
	return cr.RelatedID + "," + cr.ConfigID + cr.SelectorID
}

func (cr ConfigRelationship) TableName() string {
	return "config_relationships"
}

// ConfigChange represents the config change database table
type ConfigChange struct {
	ExternalID       string     `gorm:"-" json:"-"`
	ConfigType       string     `gorm:"-" json:"-"`
	ExternalChangeId string     `gorm:"column:external_change_id" json:"external_change_id"`
	ID               string     `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	ConfigID         string     `gorm:"column:config_id;default:''" json:"config_id"`
	ChangeType       string     `gorm:"column:change_type" json:"change_type" faker:"oneof:  RunInstances, diff" `
	Severity         Severity   `gorm:"column:severity" json:"severity"  faker:"oneof: critical, high, medium, low, info"`
	Source           string     `gorm:"column:source" json:"source"`
	Summary          string     `gorm:"column:summary;default:null" json:"summary,omitempty"`
	Patches          string     `gorm:"column:patches;default:null" json:"patches,omitempty"`
	Diff             string     `gorm:"column:diff;default:null" json:"diff,omitempty"`
	Details          types.JSON `gorm:"column:details" json:"details,omitempty"`
	CreatedAt        *time.Time `gorm:"column:created_at" json:"created_at"`
	// IsPushed when set to true indicates that the check status has been pushed to upstream.
	IsPushed bool `json:"is_pushed,omitempty"`
}

func (t ConfigChange) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigChange
	err := db.Select("config_changes.*").
		Joins("LEFT JOIN config_items ON config_items.id = config_changes.config_id").
		Where("config_items.agent_id = ?", uuid.Nil).
		Where("config_changes.is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i ConfigChange, _ int) DBTable { return i }), err
}

func (c ConfigChange) PK() string {
	return c.ID
}

func (c ConfigChange) TableName() string {
	return "config_changes"
}

func (c ConfigChange) GetExternalID() ExternalID {
	return ExternalID{
		ExternalID: []string{c.ExternalID},
		ConfigType: c.ConfigType,
	}
}

func (c ConfigChange) String() string {
	return fmt.Sprintf("[%s/%s] %s", c.ConfigType, c.ExternalID, c.ChangeType)
}

// BeforeCreate is a user defined hook for Gorm.
// It will be called when creating a record.
func (c *ConfigChange) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}

	tx.Statement.AddClause(clause.OnConflict{DoNothing: true})
	return nil
}

type ConfigAnalysis struct {
	ID            uuid.UUID     `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	ExternalID    string        `gorm:"-"`
	ConfigType    string        `gorm:"-"`
	ConfigID      uuid.UUID     `gorm:"column:config_id;default:''" json:"config_id"`
	ScraperID     *uuid.UUID    `gorm:"column:scraper_id;default:null" json:"scraper_id"`
	Analyzer      string        `gorm:"column:analyzer" json:"analyzer" faker:"oneof: ec2-instance-no-public-ip, eks-endpoint-no-public-access"`
	Message       string        `gorm:"column:message" json:"message"`
	Summary       string        `gorm:"column:summary;default:null" json:"summary,omitempty"`
	Status        string        `gorm:"column:status;default:null" json:"status,omitempty" faker:"oneof: open, resolved, silenced"`
	Severity      Severity      `gorm:"column:severity" json:"severity" faker:"oneof: critical, high, medium, low, info"`
	AnalysisType  AnalysisType  `gorm:"column:analysis_type" json:"analysis_type" faker:"oneof: availability, compliance, cost, security, performance"`
	Analysis      types.JSONMap `gorm:"column:analysis" json:"analysis,omitempty"`
	Source        string        `gorm:"column:source" json:"source,omitempty"`
	FirstObserved *time.Time    `gorm:"column:first_observed;<-:false" json:"first_observed"`
	LastObserved  *time.Time    `gorm:"column:last_observed" json:"last_observed"`
	// IsPushed when set to true indicates that the check status has been pushed to upstream.
	IsPushed bool `json:"is_pushed,omitempty"`
}

func (ConfigAnalysis) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigAnalysis
	err := db.Select("config_analysis.*").
		Joins("LEFT JOIN config_items ON config_items.id = config_analysis.config_id").
		Where("config_items.agent_id = ?", uuid.Nil).
		Where("config_analysis.is_pushed IS FALSE").
		Find(&items).Error
	return lo.Map(items, func(i ConfigAnalysis, _ int) DBTable { return i }), err
}

func (a ConfigAnalysis) PK() string {
	return a.ID.String()
}

func (a ConfigAnalysis) TableName() string {
	return "config_analysis"
}

func (a ConfigAnalysis) String() string {
	return fmt.Sprintf("[%s/%s] %s", a.ConfigType, a.ExternalID, a.Analyzer)
}

type ExternalID struct {
	ConfigType string
	ExternalID []string
}

func (e ExternalID) String() string {
	return fmt.Sprintf("%s/%s", e.ConfigType, strings.Join(e.ExternalID, ","))
}

func (e ExternalID) IsEmpty() bool {
	return e.ConfigType == "" && len(e.ExternalID) == 0
}

func (e ExternalID) CacheKey() string {
	return fmt.Sprintf("external_id:%s:%s", e.ConfigType, strings.Join(e.ExternalID, ","))
}

func (e ExternalID) WhereClause(db *gorm.DB) *gorm.DB {
	return db.Where("type = ? AND external_id  @> ?", e.ConfigType, pq.StringArray(e.ExternalID))
}

type RelatedConfigType string

const (
	RelatedConfigTypeIncoming RelatedConfigType = "incoming"
	RelatedConfigTypeOutgoing RelatedConfigType = "outgoing"
)

type RelatedConfig struct {
	Relation      string              `json:"relation"`
	RelationType  RelatedConfigType   `json:"relation_type"`
	ID            uuid.UUID           `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Tags          types.JSONStringMap `json:"tags"`
	Changes       types.JSON          `json:"changes,omitempty"`
	Analysis      types.JSON          `json:"analysis,omitempty"`
	CostPerMinute *float64            `json:"cost_per_minute,omitempty"`
	CostTotal1d   *float64            `json:"cost_total_1d,omitempty"`
	CostTotal7d   *float64            `json:"cost_total_7d,omitempty"`
	CostTotal30d  *float64            `json:"cost_total_30d,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	AgentID       uuid.UUID           `json:"agent_id"`
}
