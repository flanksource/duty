package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConfigItem represents the config item database table
type ConfigItem struct {
	ID            uuid.UUID            `gorm:"primaryKey;unique_index;not null;column:id" json:"id" faker:"uuid_hyphenated"  `
	ScraperID     *string              `gorm:"column:scraper_id;default:null" json:"scraper_id,omitempty"  `
	ConfigType    string               `gorm:"column:config_type;default:''" json:"config_type" faker:"oneof:  File, EC2Instance, KubernetesPod" `
	ExternalID    pq.StringArray       `gorm:"column:external_id;type:[]text" json:"external_id,omitempty" faker:"external_id"  `
	ExternalType  *string              `gorm:"column:external_type;default:null" json:"external_type,omitempty" faker:"oneof:  File, EC2Instance, KubernetesPod"  `
	Name          *string              `gorm:"column:name;default:null" json:"name,omitempty" faker:"name"  `
	Namespace     *string              `gorm:"column:namespace;default:null" json:"namespace,omitempty"  faker:"oneof: default, demo, prod, staging" `
	Description   *string              `gorm:"column:description;default:null" json:"description,omitempty"  `
	Config        *string              `gorm:"column:config;default:null" json:"config,omitempty"  `
	Source        *string              `gorm:"column:source;default:null" json:"source,omitempty"  `
	ParentID      *uuid.UUID           `gorm:"column:parent_id;default:null" json:"parent_id,omitempty" faker:"-"`
	Path          string               `gorm:"column:path;default:null" json:"path,omitempty" faker:"-"`
	CostPerMinute float64              `gorm:"column:cost_per_minute;default:null" json:"cost_per_minute,omitempty"`
	CostTotal1d   float64              `gorm:"column:cost_total_1d;default:null" json:"cost_total_1d,omitempty"`
	CostTotal7d   float64              `gorm:"column:cost_total_7d;default:null" json:"cost_total_7d,omitempty"`
	CostTotal30d  float64              `gorm:"column:cost_total_30d;default:null" json:"cost_total_30d,omitempty"`
	Tags          *types.JSONStringMap `gorm:"column:tags;default:null" json:"tags,omitempty"   faker:"tags"`
	CreatedAt     time.Time            `gorm:"column:created_at" json:"created_at"   `
	UpdatedAt     time.Time            `gorm:"column:updated_at" json:"updated_at"   `
}

func (c ConfigItem) TableName() string {
	return "config_items"
}

func (ci *ConfigItem) SetParent(parent *ConfigItem) {
	ci.ParentID = &parent.ID
	ci.Path = parent.Path + "." + ci.ID.String()
}

func (ci ConfigItem) String() string {
	return fmt.Sprintf("%s/%s", ci.ConfigType, ci.ID)
}

func (ci ConfigItem) ConfigJSONStringMap() (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal([]byte(*ci.Config), &m)
	return m, err
}

// ConfigScraper represents the config_scrapers database table
type ConfigScraper struct {
	ID          uuid.UUID  `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Spec        string     `json:"spec,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (c ConfigScraper) TableName() string {
	return "config_scrapers"
}

// BeforeCreate GORM hook
func (cs *ConfigScraper) BeforeCreate(tx *gorm.DB) error {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}

	return nil
}

type ConfigRelationship struct {
	ConfigID   string     `gorm:"column:config_id" json:"config_id"`
	RelatedID  string     `gorm:"column:related_id" json:"related_id"`
	Relation   string     `gorm:"column:relation" json:"relation"`
	SelectorID string     `gorm:"selector_id" json:"selector_id"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at,omitempty"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"updated_at,omitempty"`
	DeletedAt  *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (cr ConfigRelationship) TableName() string {
	return "config_relationships"
}

// ConfigChange represents the config change database table
type ConfigChange struct {
	ExternalID       string     `gorm:"-"`
	ExternalType     string     `gorm:"-"`
	ExternalChangeId string     `gorm:"column:external_change_id" json:"external_change_id"`
	ID               string     `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	ConfigID         string     `gorm:"column:config_id;default:''" json:"config_id"`
	ChangeType       string     `gorm:"column:change_type" json:"change_type" faker:"oneof:  RunInstances, diff" `
	Severity         string     `gorm:"column:severity" json:"severity"  faker:"oneof: critical, high, medium, low, info"`
	Source           string     `gorm:"column:source" json:"source"`
	Summary          string     `gorm:"column:summary;default:null" json:"summary,omitempty"`
	Patches          string     `gorm:"column:patches;default:null" json:"patches,omitempty"`
	Diff             string     `gorm:"column:diff;default:null" json:"diff,omitempty"`
	Details          types.JSON `gorm:"column:details" json:"details,omitempty"`
	CreatedAt        *time.Time `gorm:"column:created_at" json:"created_at"`
}

func (c ConfigChange) TableName() string {
	return "config_changes"
}

func (c ConfigChange) GetExternalID() ExternalID {
	return ExternalID{
		ExternalID:   []string{c.ExternalID},
		ExternalType: c.ExternalType,
	}
}

func (c ConfigChange) String() string {
	return fmt.Sprintf("[%s/%s] %s", c.ExternalType, c.ExternalID, c.ChangeType)
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
	ID            uuid.UUID           `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	ExternalID    string              `gorm:"-"`
	ExternalType  string              `gorm:"-"`
	ConfigID      uuid.UUID           `gorm:"column:config_id;default:''" json:"config_id"`
	Analyzer      string              `gorm:"column:analyzer" json:"analyzer" faker:"oneof: ec2-instance-no-public-ip, eks-endpoint-no-public-access"`
	Message       string              `gorm:"column:message" json:"message"`
	Summary       string              `gorm:"column:summary;default:null" json:"summary,omitempty"`
	Status        string              `gorm:"column:status;default:null" json:"status,omitempty" faker:"oneof: open, resolved, silenced"`
	Severity      string              `gorm:"column:severity" json:"severity" faker:"oneof: critical, high, medium, low, info"`
	AnalysisType  string              `gorm:"column:analysis_type" json:"change_type" faker:"oneof: availability, compliance, cost, security, performance"`
	Analysis      types.JSONStringMap `gorm:"column:analysis" json:"analysis,omitempty"`
	Source        string              `gorm:"column:source" json:"source,omitempty"`
	FirstObserved *time.Time          `gorm:"column:first_observed;<-:false" json:"first_observed"`
	LastObserved  *time.Time          `gorm:"column:last_observed" json:"last_observed"`
}

func (a ConfigAnalysis) TableName() string {
	return "config_analysis"
}

func (a ConfigAnalysis) String() string {
	return fmt.Sprintf("[%s/%s] %s", a.ExternalType, a.ExternalID, a.Analyzer)
}

type ExternalID struct {
	ExternalType string
	ExternalID   []string
}

func (e ExternalID) String() string {
	return fmt.Sprintf("%s/%s", e.ExternalType, strings.Join(e.ExternalID, ","))
}

func (e ExternalID) IsEmpty() bool {
	return e.ExternalType == "" && len(e.ExternalID) == 0
}

func (e ExternalID) CacheKey() string {
	return fmt.Sprintf("external_id:%s:%s", e.ExternalType, strings.Join(e.ExternalID, ","))
}

func (e ExternalID) WhereClause(db *gorm.DB) *gorm.DB {
	return db.Where("external_type = ? AND external_id  @> ?", e.ExternalType, pq.StringArray(e.ExternalID))
}
