package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/hash"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// +kubebuilder:object:generate=true
type ResourceSelector struct {
	// Agent can be the agent id or the name of the agent.
	//  Additionally, the special "self" value can be used to select resources without an agent.
	Agent string `yaml:"agent,omitempty" json:"agent,omitempty"`

	// Cache directives
	//  'no-cache' (should not fetch from cache but can be cached)
	//  'no-store' (should not cache)
	//  'max-age=X' (cache for X duration)
	Cache string `yaml:"cache,omitempty" json:"cache,omitempty"`

	IncludeDeleted bool `yaml:"-" json:"-"`

	ID            string `yaml:"id,omitempty" json:"id,omitempty"`
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace     string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Types         Items  `yaml:"types,omitempty" json:"types,omitempty"`
	Statuses      Items  `yaml:"statuses,omitempty" json:"statuses,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`
}

func (c ResourceSelector) IsEmpty() bool {
	return c.ID == "" && c.Name == "" && c.Namespace == "" && c.Agent == "" && len(c.Types) == 0 &&
		len(c.Statuses) == 0 &&
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

	if c.Namespace == "" {
		return false // still not specific enough
	}

	if len(c.LabelSelector) != 0 || len(c.FieldSelector) != 0 || len(c.Statuses) != 0 {
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
		strings.Join(c.Types.Sort(), ","),
		strings.Join(c.Statuses.Sort(), ","),
		collections.SortedMap(collections.SelectorToMap(c.LabelSelector)),
		collections.SortedMap(collections.SelectorToMap(c.FieldSelector)),
		fmt.Sprint(c.IncludeDeleted),
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
	if len(rs.Statuses) > 0 && !rs.Statuses.Contains(s.GetStatus()) {
		return false
	}

	if len(rs.LabelSelector) > 0 {
		for k, v := range collections.SelectorToMap(rs.LabelSelector) {
			if sVal, exists := s.GetLabels()[k]; exists {
				if v != "" && v != sVal {
					return false
				}
			} else {
				return false
			}
		}
	}

	if len(rs.FieldSelector) > 0 {
		for k, v := range collections.SelectorToMap(rs.FieldSelector) {
			if sVal, exists := s.GetFields()[k]; exists {
				if v != "" && v != sVal {
					return false
				}
			} else {
				return false
			}
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

type ResourceSelectable interface {
	GetID() string
	GetName() string
	GetNamespace() string
	GetType() string
	GetStatus() string
	GetLabels() map[string]string
	GetFields() map[string]string
}
