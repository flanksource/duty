package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type ComponentConfigTraversalArgs struct {
	ComponentID string `yaml:"component_id,omitempty" json:"component_id,omitempty"`
	Direction   string `yaml:"direction,omitempty" json:"direction,omitempty"`
}

// +kubebuilder:object:generate=true
type Functions struct {
	// It uses the config_id linked to the componentID to lookup up all the config relations and returns
	// a list of componentIDs that are linked to the found configIDs
	ComponentConfigTraversal *ComponentConfigTraversalArgs `yaml:"component_config_traversal,omitempty" json:"component_config_traversal,omitempty"`
}

type QueryOperator string

const (
	Eq  QueryOperator = "="
	Neq QueryOperator = "!="

	Gt        QueryOperator = ">"
	Lt        QueryOperator = "<"
	In        QueryOperator = "in"
	NotIn     QueryOperator = "notin"
	Exists    QueryOperator = "exists"
	NotExists QueryOperator = "!"
)

func (op QueryOperator) ToSelectionOperator() selection.Operator {
	switch op {
	case Eq:
		return selection.Equals
	case Neq:
		return selection.NotEquals
	case In:
		return selection.In
	case NotIn:
		return selection.NotIn
	case Exists:
		return selection.Exists
	case NotExists:
		return selection.DoesNotExist
	default:
		return selection.Equals
	}
}

type QueryField struct {
	Field  string        `json:"field,omitempty"`
	Value  interface{}   `json:"value,omitempty"`
	Op     QueryOperator `json:"op,omitempty"`
	Not    bool          `json:"not,omitempty"`
	Fields []*QueryField `json:"fields,omitempty"`
}

var CommonFields = map[string]bool{
	"id":   true,
	"type": true,
}

func (f *QueryField) ToLabelSelector() (labels.Selector, error) {
	selector := labels.NewSelector()
	for _, field := range f.Fields {
		if CommonFields[field.Field] {
			continue
		}
		val := fmt.Sprintf("%s", field.Value)
		req, err := labels.NewRequirement(field.Field, field.Op.ToSelectionOperator(), []string{val})
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*req)
	}
	return selector, nil
}

// +kubebuilder:object:generate=true
type ResourceSelector struct {
	// Agent can be the agent id or the name of the agent.
	//  Additionally, the special "self" value can be used to select resources without an agent.
	Agent string `yaml:"agent,omitempty" json:"agent,omitempty"`

	// Scope is the reference for parent of the resource to select.
	// For config items, the scope is the scraper id
	// For checks, it's canaries and
	// For components, it's topology.
	// It can either be a uuid or namespace/name
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`

	// Cache directives
	//  'no-cache' (should not fetch from cache but can be cached)
	//  'no-store' (should not cache)
	//  'max-age=X' (cache for X duration)
	Cache string `yaml:"cache,omitempty" json:"cache,omitempty"`

	// Search query that applies to the resource name, tag & labels.
	Search string `yaml:"search,omitempty" json:"search,omitempty" template:"true"`

	// Use custom functions for specific selections
	Functions Functions `yaml:"-" json:"-"`

	Limit int `yaml:"limit,omitempty" json:"limit,omitempty"`

	IncludeDeleted bool `yaml:"includeDeleted,omitempty" json:"includeDeleted,omitempty"`

	ID            string `yaml:"id,omitempty" json:"id,omitempty"`
	Name          string `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace     string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	TagSelector   string `yaml:"tagSelector,omitempty" json:"tagSelector,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`

	// Health filters resources by the health.
	// Multiple healths can be provided separated by comma.
	Health MatchExpression `json:"health,omitempty"`

	// Types filter resources by the type
	Types Items `yaml:"types,omitempty" json:"types,omitempty"`

	// Statuses filter resources by the status
	Statuses Items `yaml:"statuses,omitempty" json:"statuses,omitempty"`
}

// ParseFilteringQuery parses a filtering query string.
// It returns four slices: 'in', 'notIN', 'prefix', and 'suffix'.
func ParseFilteringQuery(query string, decodeURL bool) (in []interface{}, notIN []interface{}, prefix, suffix []string, err error) {
	if query == "" {
		return
	}

	items := strings.Split(query, ",")
	for _, item := range items {
		if decodeURL {
			item, err = url.QueryUnescape(item)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to unescape query (%s): %v", item, err)
			}
		}

		if strings.HasPrefix(item, "!") {
			notIN = append(notIN, strings.TrimPrefix(item, "!"))
		} else if strings.HasPrefix(item, "*") {
			suffix = append(suffix, strings.TrimPrefix(item, "*"))
		} else if strings.HasSuffix(item, "*") {
			prefix = append(prefix, strings.TrimSuffix(item, "*"))
		} else {
			in = append(in, item)
		}
	}

	return
}

func (q QueryField) ToClauses() ([]clause.Expression, error) {
	val := fmt.Sprint(q.Value)

	filters, err := ParseFilteringQueryV2(val, false)
	if err != nil {
		return nil, err
	}

	var clauses []clause.Expression
	switch q.Op {
	case Eq:
		clauses = append(clauses, filters.ToExpression(q.Field)...)
	case Neq:
		clauses = append(clauses, clause.Not(filters.ToExpression(q.Field)...))
	case Lt:
		clauses = append(clauses, clause.Lt{Column: q.Field, Value: q.Value})
	case Gt:
		clauses = append(clauses, clause.Gt{Column: q.Field, Value: q.Value})
	default:
		return nil, fmt.Errorf("invalid operator: %s", q.Op)
	}

	return clauses, nil
}

func (c ResourceSelector) allEmptyButName() bool {
	return c.ID == "" && c.Namespace == "" && c.Agent == "" && c.Scope == "" && c.Search == "" &&
		len(c.Types) == 0 &&
		len(c.Statuses) == 0 &&
		len(c.Health) == 0 &&
		len(c.TagSelector) == 0 &&
		len(c.LabelSelector) == 0 &&
		len(c.FieldSelector) == 0
}

// A wildcard resource selector is one where it just has the name field set to '*'
func (c ResourceSelector) Wildcard() bool {
	return c.allEmptyButName() && c.Name == "*"
}

func (c ResourceSelector) IsEmpty() bool {
	return c.allEmptyButName() && c.Name == ""
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

	if len(c.TagSelector) != 0 || len(c.LabelSelector) != 0 || len(c.FieldSelector) != 0 || len(c.Statuses) != 0 || len(c.Health) != 0 {
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
		string(c.Health),
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
	if rs.Wildcard() {
		return true
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

	if h, err := s.GetHealth(); err != nil {
		logger.Errorf("failed to get health: %v", err)
		return false
	} else if len(rs.Health) > 0 && !rs.Health.Match(h) {
		return false
	}

	if len(rs.TagSelector) > 0 {
		if tagsMatcher, ok := s.(TagsMatchable); ok {
			parsed, err := labels.Parse(rs.TagSelector)
			if err != nil {
				logger.Errorf("bad tag selector: %v", err)
				return false
			} else if !parsed.Matches(tagsMatcher.GetTagsMatcher()) {
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

// MatchSelectables returns only those selectables that have at matches with at least one of the given selectors.
func MatchSelectables[T ResourceSelectable](selectables []T, selectors ...ResourceSelector) []T {
	if len(selectors) == 0 {
		return nil
	}

	var matches []T
	for _, selectable := range selectables {
		for _, selector := range selectors {
			if selector.Matches(selectable) {
				matches = append(matches, selectable)
				break
			}
		}
	}

	return matches
}

type TagsMatchable interface {
	GetTagsMatcher() labels.Labels
}

type ResourceSelectable interface {
	GetFieldsMatcher() fields.Fields
	GetLabelsMatcher() labels.Labels

	GetID() string
	GetName() string
	GetNamespace() string
	GetType() string
	GetStatus() (string, error)
	GetHealth() (string, error)
}
