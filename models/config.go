package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/hash"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/flanksource/duty/types"
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

type RelatedConfigDirection string

const (
	RelatedConfigTypeIncoming RelatedConfigDirection = "incoming"
	RelatedConfigTypeOutgoing RelatedConfigDirection = "outgoing"
)

// Ensure interface compliance
var (
	_ types.ResourceSelectable = ConfigItem{}
	_ types.TagsMatchable      = ConfigItem{}
	_ TaggableModel            = ConfigItem{}
	_ LabelableModel           = ConfigItem{}
)

// ConfigLocation represents the config_locations database table
type ConfigLocation struct {
	ID       uuid.UUID `json:"id"`
	Location string    `json:"location"`
}

// ConfigItem represents the config item database table
type ConfigItem struct {
	ID            uuid.UUID            `json:"id" faker:"uuid_hyphenated" gorm:"default:generate_ulid()"`
	ScraperID     *string              `json:"scraper_id,omitempty"`
	AgentID       uuid.UUID            `json:"agent_id,omitempty"`
	ConfigClass   string               `json:"config_class" faker:"oneof:File,EC2Instance,KubernetesPod" `
	ExternalID    pq.StringArray       `gorm:"type:[]text" json:"external_id,omitempty"`
	Type          *string              `json:"type"`
	Status        *string              `json:"status" gorm:"default:null"`
	Ready         bool                 `json:"ready"`
	Health        *Health              `json:"health"`
	Name          *string              `json:"name,omitempty" faker:"name"`
	Description   *string              `json:"description"`
	Config        *string              `json:"config"`
	Source        *string              `json:"source,omitempty"`
	ParentID      *uuid.UUID           `json:"parent_id,omitempty" faker:"-"`
	Path          string               `json:"path,omitempty" faker:"-"`
	CostPerMinute float64              `gorm:"column:cost_per_minute;default:null" json:"cost_per_minute,omitempty"`
	CostTotal1d   float64              `gorm:"column:cost_total_1d;default:null" json:"cost_total_1d,omitempty"`
	CostTotal7d   float64              `gorm:"column:cost_total_7d;default:null" json:"cost_total_7d,omitempty"`
	CostTotal30d  float64              `gorm:"column:cost_total_30d;default:null" json:"cost_total_30d,omitempty"`
	Labels        *types.JSONStringMap `json:"labels,omitempty" faker:"labels"`
	Tags          types.JSONStringMap  `json:"tags,omitempty" faker:"tags"`
	Properties    *types.Properties    `json:"properties,omitempty"`
	CreatedAt     time.Time            `json:"created_at" gorm:"<-:create"`
	UpdatedAt     *time.Time           `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt     *time.Time           `json:"deleted_at,omitempty"`
	DeleteReason  string               `json:"delete_reason,omitempty"`

	configJson map[string]any `json:"-" yaml:"-" gorm:"-"`
}

type ConfigItemLastScrapedTime struct {
	ConfigID        string     `json:"config_id" gorm:"primaryKey"`
	LastScrapedTime *time.Time `json:"last_scraped_time,omitempty"`
}

func (ConfigItemLastScrapedTime) TableName() string {
	return "config_items_last_scraped_time"
}

func (ConfigItemLastScrapedTime) PK() string {
	return "config_id"
}

func (ConfigItemLastScrapedTime) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	return nil, nil
}

func (ConfigItemLastScrapedTime) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	return nil
}

// This should only be used for tests and its fixtures
func DeleteAllConfigs(db *gorm.DB, configs ...ConfigItem) error {
	ids := lo.Map(configs, func(c ConfigItem, _ int) string { return c.ID.String() })

	return db.Exec("select drop_config_items(?)", pq.StringArray(ids)).Error
}

func (t ConfigItem) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	configWithScraper := lo.Filter(items, func(item DBTable, _ int) bool { return item.(ConfigItem).ScraperID != nil })
	scraperParents := lo.Map(configWithScraper, func(item DBTable, _ int) string {
		return *item.(ConfigItem).ScraperID
	})

	if len(scraperParents) > 0 {
		if err := db.Model(&ConfigScraper{}).Where("id IN ?", scraperParents).Update("is_pushed", false).Error; err != nil {
			return err
		}
	}

	// config items can also have another config items as parent
	configWithConfigParent := lo.Filter(items, func(item DBTable, _ int) bool { return item.(ConfigItem).ParentID != nil })
	configParents := lo.Map(configWithConfigParent, func(item DBTable, _ int) string {
		return item.(ConfigItem).ParentID.String()
	})
	if len(configParents) > 0 {
		if err := db.Model(&ConfigItem{}).Where("id IN ?", configParents).Update("is_pushed", false).Error; err != nil {
			return err
		}
	}

	return nil
}

func (t ConfigItem) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigItem
	err := db.Where("is_pushed IS FALSE").Order("LENGTH(COALESCE(path, ''))").Find(&items).Error
	return lo.Map(items, func(i ConfigItem, _ int) DBTable { return i }), err
}

func (t ConfigItem) PK() string {
	return t.ID.String()
}

func (t ConfigItem) PKCols() []clause.Column {
	return []clause.Column{{Name: "id"}}
}

func (c ConfigItem) Value() any {
	return &c
}

func (ConfigItem) TableName() string {
	return "config_items"
}

func (t ConfigItem) GetTags() map[string]string {
	return t.Tags
}

func (t ConfigItem) GetLabels() map[string]string {
	return lo.FromPtr(t.Labels)
}

func (t ConfigItem) GetTrimmedLabels() []Label {
	return sortedTrimmedLabels(defaultLabelsWhitelist, defaultLabelsOrder, t.Tags, lo.FromPtr(t.Labels))
}

func (ci *ConfigItem) SetParent(parent *ConfigItem) {
	ci.ParentID = &parent.ID
	ci.Path = parent.Path + "." + ci.ID.String()
}

func (ci ConfigItem) String() string {
	return fmt.Sprintf("%s{name=%s, id=%s}", ci.ConfigClass, *ci.Name, ci.ID)
}

func (ci ConfigItem) AsMap(removeFields ...string) map[string]any {
	env := asMap(ci, removeFields...)
	if ci.Config == nil || *ci.Config == "" {
		return env
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(*ci.Config), &m); err != nil {
		return env
	}
	env["config"] = m

	return env
}

func (ci *ConfigItem) FromMap(data map[string]any) error {
	if configValue, exists := data["config"]; exists && configValue != nil {
		switch v := configValue.(type) {
		case string:
			ci.Config = &v
		default:
			configBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal config map to JSON: %w", err)
			}
			ci.Config = lo.ToPtr(string(configBytes))
		}

		// Config is directly set to the model, so we don't need to unmarshal it again
		delete(data, "config")
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal map data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, ci); err != nil {
		return fmt.Errorf("failed to unmarshal data into ConfigItem: %w", err)
	}

	return nil
}

func (ci *ConfigItem) ConfigJSONStringMap() (map[string]any, error) {
	if ci.configJson != nil {
		return ci.configJson, nil
	}
	ci.configJson = make(map[string]any)
	err := json.Unmarshal([]byte(*ci.Config), &ci.configJson)
	return ci.configJson, err
}

func (ci *ConfigItem) NestedString(paths ...string) string {
	m, err := ci.ConfigJSONStringMap()
	if err != nil {
		return ""
	}
	v, _, _ := unstructured.NestedString(m, paths...)
	return v
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
	return c.Tags["namespace"]
}

func (c ConfigItem) GetType() string {
	if c.Type == nil {
		return ""
	}
	return *c.Type
}

func (c ConfigItem) GetHealth() (string, error) {
	return string(lo.FromPtr(c.Health)), nil
}

func (c ConfigItem) GetStatus() (string, error) {
	if c.Status == nil {
		return "", nil
	}
	return *c.Status, nil
}

func (c ConfigItem) GetAgentID() string {
	if c.AgentID == uuid.Nil {
		return ""
	}
	return c.AgentID.String()
}

func (c ConfigItem) GetHealthDescription() string {
	return lo.FromPtr(c.Description)
}

func (c ConfigItem) GetTagsMatcher() labels.Labels {
	return types.GenericLabelsMatcher{Map: c.Tags}
}

func (c ConfigItem) GetLabelsMatcher() labels.Labels {
	return configLabels{c}
}

func (c ConfigItem) GetFieldsMatcher() fields.Fields {
	return types.GenericFieldMatcher{Fields: c.AsMap()}
}

type configLabels struct {
	ConfigItem
}

func (c configLabels) Get(key string) string {
	if c.Labels == nil || len(*c.Labels) == 0 {
		return ""
	}

	return (*c.Labels)[key]
}

func (c configLabels) Has(key string) bool {
	if c.Labels == nil || len(*c.Labels) == 0 {
		return false
	}

	_, ok := (*c.Labels)[key]
	return ok
}

func (c configLabels) Lookup(key string) (string, bool) {
	if c.Labels == nil || len(*c.Labels) == 0 {
		return "", false
	}

	value, ok := (*c.Labels)[key]
	return value, ok
}

// ConfigScraper represents the config_scrapers database table
type ConfigScraper struct {
	ID            uuid.UUID  `json:"id"`
	AgentID       uuid.UUID  `json:"agent_id,omitempty"`
	Name          string     `json:"name"`
	Namespace     string     `json:"namespace"`
	Description   string     `json:"description,omitempty"`
	Spec          string     `json:"spec,omitempty"`
	Source        string     `json:"source,omitempty"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty" gorm:"default:null"`
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at" gorm:"<-:create"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

func FindScraperByConfigId(db *gorm.DB, configId string) (*ConfigScraper, error) {
	var configItem ConfigItem
	if err := db.Where("id = ?", configId).Find(&configItem).Error; err != nil {
		return nil, fmt.Errorf("failed to get config (%s): %w", configId, err)
	} else if configItem.ID == uuid.Nil {
		return nil, fmt.Errorf("config item not found: %s", configId)
	}

	if lo.FromPtr(configItem.ScraperID) == "" {
		return nil, fmt.Errorf("config item does not have a scraper: %s", configId)
	}

	var scrapeConfig ConfigScraper
	if err := db.Where("id = ?", lo.FromPtr(configItem.ScraperID)).Find(&scrapeConfig).Error; err != nil {
		return nil, fmt.Errorf("failed to get scrapeconfig (%s): %w", lo.FromPtr(configItem.ScraperID), err)
	} else if scrapeConfig.ID.String() != lo.FromPtr(configItem.ScraperID) {
		return nil, fmt.Errorf("scraper not found: %s", lo.FromPtr(configItem.ScraperID))
	}

	return &scrapeConfig, nil
}

func (c ConfigScraper) GetNamespace() string {
	return c.Namespace
}

func (c ConfigScraper) GetAgentID() string {
	if c.AgentID == uuid.Nil {
		return ""
	}
	return c.AgentID.String()
}

func (c ConfigScraper) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigScraper
	err := db.Where("is_pushed IS FALSE AND id != ?", uuid.Nil).Find(&items).Error
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
	ConfigID   string     `json:"config_id" gorm:"primaryKey"`
	RelatedID  string     `json:"related_id" gorm:"primaryKey"`
	Relation   string     `json:"relation" gorm:"primaryKey"`
	SelectorID string     `json:"selector_id"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty" gorm:"autoUpdateTime:false"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func (c ConfigRelationship) Value() any {
	return &c
}

func (c ConfigRelationship) PKCols() []clause.Column {
	return []clause.Column{{Name: "related_id"}, {Name: "config_id"}, {Name: "relation"}}
}

func (t ConfigRelationship) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(ConfigRelationship).ConfigID
	})

	relatedIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(ConfigRelationship).RelatedID
	})

	return db.Model(&ConfigItem{}).Where("id IN ?", append(parentIDs, relatedIDs...)).Update("is_pushed", false).Error
}

func (s ConfigRelationship) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	ids := lo.Map(items, func(a DBTable, _ int) []string {
		c := any(a).(ConfigRelationship)
		return []string{c.RelatedID, c.ConfigID, c.Relation}
	})

	return db.Model(&ConfigRelationship{}).Where("(related_id, config_id, relation) IN ?", ids).Update("is_pushed", true).Error
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
// ConfigChange represents a change to a configuration item.
type ConfigChange struct {
	// ExternalID is the external identifier for the configuration change.
	// Note: This field is not stored in the database.
	ExternalID string `gorm:"-" json:"-"`

	// ConfigType represents the type of configuration.
	// Note: This field is not stored in the database.
	ConfigType string `gorm:"-" json:"-"`

	// ExternalChangeID is the identifier for the change from an external system.
	ExternalChangeID *string `gorm:"column:external_change_id;default:null" json:"external_change_id"`

	// ID is the unique identifier for the configuration change.
	// It is automatically generated using ULID if not provided.
	ID string `gorm:"primaryKey;unique_index;not null;column:id;default:generate_ulid()" json:"id"`

	// ConfigID is the identifier of the associated configuration item.
	ConfigID string `gorm:"column:config_id;default:''" json:"config_id"`

	// ChangeType describes the nature of the configuration change.
	// Example values: RunInstances, diff
	ChangeType string `gorm:"column:change_type" json:"change_type" faker:"oneof:  RunInstances, diff" `

	// Severity indicates the importance or impact level of the change.
	// Possible values: critical, high, medium, low, info
	Severity Severity `json:"severity"  faker:"oneof: critical, high, medium, low, info"`

	// Source indicates the origin of the configuration change, e.g. Kubernetes, Cloudtrail
	Source string `json:"source"`

	// Summary provides a brief description of the change.
	Summary string `json:"summary,omitempty"`

	// Patches contains a JSON strategic merge patch
	Patches string `gorm:"column:patches;default:null" json:"patches,omitempty"`

	// Diff represents the differences introduced by this change.
	Diff string `gorm:"column:diff;default:null" json:"diff,omitempty"`

	// Fingerprint is a uniquest identifier for the change, it ignores all UUID, numbers and timestamps to enable de-duplication of equivalent changes.
	Fingerprint string `gorm:"column:fingerprint;default:null" json:"fingerprint,omitempty"`

	// Details contains additional information about the change in JSON format.
	Details types.JSON `json:"details,omitempty"`

	// CreatedAt represents the timestamp when the change was created or last observed
	CreatedAt *time.Time `json:"created_at"`

	// FirstObserved represents the timestamp when this change was first observed.
	FirstObserved *time.Time `gorm:"first_observed;default:now()" json:"first_observed,omitempty"`

	// Count is the number of occurrences of this change, including duplicates detected by fingerprinting
	Count int `json:"count"`

	// IsPushed indicates whether the change has been pushed to upstream.
	// When set to true, it means the status has been synchronized.
	IsPushed bool `json:"is_pushed,omitempty"`
}

func (t ConfigChange) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(ConfigChange).ConfigID
	})

	return db.Model(&ConfigItem{}).Where("id IN ?", parentIDs).Update("is_pushed", false).Error
}

func (t ConfigChange) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []ConfigChange
	err := db.Select("config_changes.*").
		Joins("LEFT JOIN config_items ON config_items.id = config_changes.config_id").
		Where("config_items.agent_id = ?", uuid.Nil).
		Where("config_changes.is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i ConfigChange, _ int) DBTable { return i }), err
}

func (c ConfigChange) PKCols() []clause.Column {
	return []clause.Column{{Name: "id"}}
}

func (c ConfigChange) Value() any {
	return &c
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
	ID            uuid.UUID     `gorm:"primaryKey;unique_index;not null;column:id;default:generate_ulid()" json:"id"`
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

func (t ConfigAnalysis) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(ConfigAnalysis).ConfigID.String()
	})

	return db.Model(&ConfigItem{}).Where("id IN ?", parentIDs).Update("is_pushed", false).Error
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

func (c ConfigAnalysis) PKCols() []clause.Column {
	return []clause.Column{{Name: "id"}}
}

func (c ConfigAnalysis) Value() any {
	return &c
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

// ConfigItemSummary represents the configs view
type ConfigItemSummary struct {
	ID            uuid.UUID            `json:"id" gorm:"primaryKey"`
	ScraperID     *string              `json:"scraper_id,omitempty"`
	ConfigClass   string               `json:"config_class"`
	ExternalID    pq.StringArray       `gorm:"type:[]text" json:"external_id,omitempty"`
	Type          *string              `json:"type"`
	Name          *string              `json:"name,omitempty"`
	Namespace     *string              `json:"namespace,omitempty"`
	Description   *string              `json:"description"`
	Source        *string              `json:"source,omitempty"`
	Labels        *types.JSONStringMap `json:"labels,omitempty"`
	Tags          types.JSONStringMap  `json:"tags,omitempty"`
	CreatedBy     *uuid.UUID           `json:"created_by,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     *time.Time           `json:"updated_at"`
	DeletedAt     *time.Time           `json:"deleted_at,omitempty"`
	CostPerMinute float64              `gorm:"column:cost_per_minute" json:"cost_per_minute,omitempty"`
	CostTotal1d   float64              `gorm:"column:cost_total_1d" json:"cost_total_1d,omitempty"`
	CostTotal7d   float64              `gorm:"column:cost_total_7d" json:"cost_total_7d,omitempty"`
	CostTotal30d  float64              `gorm:"column:cost_total_30d" json:"cost_total_30d,omitempty"`
	AgentID       uuid.UUID            `json:"agent_id,omitempty"`
	Status        *string              `json:"status"`
	Health        *Health              `json:"health"`
	Ready         bool                 `json:"ready"`
	Path          string               `json:"path,omitempty"`
	Changes       int                  `json:"changes,omitempty"`
	Analysis      *types.JSONMap       `json:"analysis,omitempty"`
}

func (ConfigItemSummary) TableName() string {
	return "configs"
}

func (c ConfigItemSummary) GetAgentID() string {
	if c.AgentID == uuid.Nil {
		return ""
	}
	return c.AgentID.String()
}

func (c ConfigItemSummary) ToConfigItem() ConfigItem {
	return ConfigItem{
		ID:            c.ID,
		ScraperID:     c.ScraperID,
		AgentID:       c.AgentID,
		ConfigClass:   c.ConfigClass,
		ExternalID:    c.ExternalID,
		Type:          c.Type,
		Status:        c.Status,
		Ready:         c.Ready,
		Health:        c.Health,
		Name:          c.Name,
		Description:   c.Description,
		Source:        c.Source,
		Path:          c.Path,
		CostPerMinute: c.CostPerMinute,
		CostTotal1d:   c.CostTotal1d,
		CostTotal7d:   c.CostTotal7d,
		CostTotal30d:  c.CostTotal30d,
		Labels:        c.Labels,
		Tags:          c.Tags,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
		DeletedAt:     c.DeletedAt,
	}
}
