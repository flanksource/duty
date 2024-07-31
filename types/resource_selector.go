package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// +kubebuilder:object:generate=true
type ResourceSelector struct {
	// Agent can be the agent id or the name of the agent.
	//  Additionally, the special "self" value can be used to select resources without an agent.
	Agent string `yaml:"agent,omitempty" json:"agent,omitempty"`

	// Scope is the id parent of the resource to select.
	// Example: For config items, the scope is the scraper id
	// - for checks, it's canaries and
	// - for components, it's topology.
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`

	// Cache directives
	//  'no-cache' (should not fetch from cache but can be cached)
	//  'no-store' (should not cache)
	//  'max-age=X' (cache for X duration)
	Cache string `yaml:"cache,omitempty" json:"cache,omitempty"`

	// Search query that applies to the resource name, tag & labels.
	Search string `yaml:"search,omitempty" json:"search,omitempty"`

	IncludeDeleted bool `yaml:"includeDeleted,omitempty" json:"includeDeleted,omitempty"`

	ID            string `yaml:"id,omitempty" json:"id,omitempty"`
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace     string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Types         Items  `yaml:"types,omitempty" json:"types,omitempty"`
	Statuses      Items  `yaml:"statuses,omitempty" json:"statuses,omitempty"`
	TagSelector   string `yaml:"tagSelector,omitempty" json:"tagSelector,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`
}

func (c ResourceSelector) IsEmpty() bool {
	return c.ID == "" && c.Name == "" && c.Namespace == "" && c.Agent == "" && c.Scope == "" && c.Search == "" &&
		len(c.Types) == 0 &&
		len(c.Statuses) == 0 &&
		len(c.TagSelector) == 0 &&
		len(c.LabelSelector) == 0 &&
		len(c.FieldSelector) == 0
}

// Immutable returns true if the selector can be cached indefinitely
func (c ResourceSelector) Immutable() bool {
	if c.ID != "" {
		return true
	}

	if c.Name == "" {
		// without a name, a selector is never specific enough to be cached indefinitely
		return false
	}

	if c.Search == "" {
		// too broad to be cached indefinitely
		return false
	}

	if c.Namespace == "" {
		return false // still not specific enough
	}

	if len(c.TagSelector) != 0 || len(c.LabelSelector) != 0 || len(c.FieldSelector) != 0 || len(c.Statuses) != 0 {
		// These selectors work on mutable part of the resource, so they can't be cached indefinitely
		return false
	}

	return true
}

func (c ResourceSelector) Hash() string {
	items := []string{
		c.ID,
		c.Name,
		c.Namespace,
		c.Agent,
		c.Scope,
		strings.Join(c.Types.Sort(), ","),
		strings.Join(c.Statuses.Sort(), ","),
		collections.SortedMap(collections.SelectorToMap(c.TagSelector)),
		collections.SortedMap(collections.SelectorToMap(c.LabelSelector)),
		collections.SortedMap(collections.SelectorToMap(c.FieldSelector)),
		fmt.Sprint(c.IncludeDeleted),
		c.Search,
	}

	return hash.Sha256Hex(strings.Join(items, "|"))
}

func (rs ResourceSelector) Matches(s ResourceSelectable) bool {
	if rs.IsEmpty() {
		return false
	}
	if rs.ID != "" && rs.ID != s.GetID() {
		return false
	}
	if rs.Name != "" && rs.Name != s.GetName() {
		return false
	}
	if rs.Namespace != "" && rs.Namespace != s.GetNamespace() {
		return false
	}
	if len(rs.Types) > 0 && !rs.Types.Contains(s.GetType()) {
		return false
	}

	if status, err := s.GetStatus(); err != nil {
		logger.Errorf("failed to get status: %v", err)
		return false
	} else if len(rs.Statuses) > 0 && !rs.Statuses.Contains(status) {
		return false
	}

	if len(rs.TagSelector) > 0 {
		if tagsMatcher, ok := s.(TagsMatchable); ok {
			parsed, err := labels.Parse(rs.TagSelector)
			if err != nil {
				logger.Errorf("bad tag selector: %v", err)
				return false
			} else if !parsed.Matches(tagsMatcher.GetTagssMatcher()) {
				return false
			}
		}
	}

	if len(rs.LabelSelector) > 0 {
		parsed, err := labels.Parse(rs.LabelSelector)
		if err != nil {
			logger.Errorf("bad label selector: %v", err)
			return false
		} else if !parsed.Matches(s.GetLabelsMatcher()) {
			return false
		}
	}

	if len(rs.FieldSelector) > 0 {
		parsed, err := labels.Parse(rs.FieldSelector)
		if err != nil {
			logger.Errorf("bad field selector: %v", err)
			return false
		} else if !parsed.Matches(s.GetFieldsMatcher()) {
			return false
		}
	}

	return true
}

type ResourceSelectors []ResourceSelector

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

type TagsMatchable interface {
	GetTagssMatcher() labels.Labels
}

type ResourceSelectable interface {
	GetFieldsMatcher() fields.Fields
	GetLabelsMatcher() labels.Labels

	GetID() string
	GetName() string
	GetNamespace() string
	GetType() string
	GetStatus() (string, error)
}
