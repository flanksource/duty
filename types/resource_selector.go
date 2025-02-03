package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/query/grammar"
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

func (rs ResourceSelector) ToPeg() string {
	var searchConditions []string

	if rs.ID != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("id = %q", rs.ID))
	}

	if rs.Name != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("name = %q", rs.Name))
	}

	if rs.Namespace != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("namespace = %q", rs.Namespace))
	}

	if len(rs.Health) != 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("health = %q", rs.Health))
	}

	if len(rs.Types) > 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("type = %q", strings.Join(rs.Types, ",")))
	}

	if len(rs.Statuses) > 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("status = %q", strings.Join(rs.Statuses, ",")))
	}

	if rs.LabelSelector != "" {
		searchConditions = append(searchConditions, selectorToPegCondition("labels.", rs.LabelSelector)...)
	}

	if rs.TagSelector != "" {
		searchConditions = append(searchConditions, selectorToPegCondition("tags.", rs.TagSelector)...)
	}

	if rs.FieldSelector != "" {
		searchConditions = append(searchConditions, selectorToPegCondition("", rs.FieldSelector)...)
	}

	peg := rs.Search
	if len(searchConditions) > 0 {
		joined := strings.Join(searchConditions, " ")
		peg += fmt.Sprintf(" %s", joined)
	}

	return peg
}

func selectorToPegCondition(fieldPrefix, selector string) []string {
	parsed, err := labels.Parse(selector)
	if err != nil {
		return nil
	}

	requirements, selectable := parsed.Requirements()
	if !selectable {
		return nil
	}

	var searchConditions []string
	for _, requirement := range requirements {
		operator := grammar.Eq

		switch requirement.Operator() {
		case selection.Equals, selection.In:
			operator = grammar.Eq
		case selection.NotEquals, selection.NotIn:
			operator = grammar.Neq
		case selection.GreaterThan:
			operator = grammar.Gt
		case selection.LessThan:
			operator = grammar.Lt
		}

		condition := fmt.Sprintf("%s%s %s %q", fieldPrefix, requirement.Key(), operator, strings.Join(requirement.Values().List(), ","))
		searchConditions = append(searchConditions, condition)
	}

	return searchConditions
}

func (rs ResourceSelector) Matches(s ResourceSelectable) bool {
	if rs.IsEmpty() {
		return false
	}

	if rs.Wildcard() {
		return true
	}

	peg := rs.ToPeg()
	if peg == "" {
		return false
	}

	qf, err := grammar.ParsePEG(peg)
	if err != nil {
		return false
	}

	return rs.matchGrammar(qf, s)
}

func (rs *ResourceSelector) matchGrammar(qf *grammar.QueryField, s ResourceSelectable) bool {
	if qf.Field != "" {
		var err error

		var value string
		switch qf.Field {
		case "name":
			value = s.GetName()
		case "namespace":
			value = s.GetNamespace()
		case "id":
			value = s.GetID()
		case "type":
			value = s.GetType()
		case "status":
			value, err = s.GetStatus()
			if err != nil {
				logger.Errorf("failed to get status: %v", err)
				return false
			}
		case "health":
			value, err = s.GetHealth()
			if err != nil {
				logger.Errorf("failed to get health: %v", err)
				return false
			}
		default:
			if strings.HasPrefix(qf.Field, "labels.") {
				key := strings.TrimSpace(strings.TrimPrefix(qf.Field, "labels."))
				value = s.GetLabelsMatcher().Get(key)
			} else if strings.HasPrefix(qf.Field, "tags.") {
				key := strings.TrimSpace(strings.TrimPrefix(qf.Field, "tags."))
				if tagsMatcher, ok := s.(TagsMatchable); ok {
					value = tagsMatcher.GetTagsMatcher().Get(key)
				}
			} else if strings.HasPrefix(qf.Field, "properties.") {
				propertyName := strings.TrimSpace(strings.TrimPrefix(qf.Field, "properties."))
				value = s.GetFieldsMatcher().Get("properties")
				var properties Properties
				if err := json.Unmarshal([]byte(value), &properties); err != nil {
					logger.Errorf("failed to unmarshall properties")
					return false
				}

				for _, p := range properties {
					if p.Name != propertyName {
						continue
					}

					if p.Text != "" {
						value = p.Text
					} else if p.Value != nil {
						value = strconv.FormatInt(*p.Value, 10)
					}
				}
			} else {
				// Unknown key is a field selector
				key := strings.TrimSpace(qf.Field)
				value = s.GetFieldsMatcher().Get(key)
			}
		}

		var patterns []string
		if qfs, ok := qf.Value.(string); ok {
			patterns = strings.Split(qfs, ",")
		}

		switch qf.Op {
		case grammar.Eq:
			return collections.MatchItems(value, patterns...)

		case grammar.Neq:
			return !collections.MatchItems(value, patterns...)

		case grammar.Gt, grammar.Lt:
			propertyValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				logger.WithValues("value", value).Errorf("properties lessthan and greaterthan operator only supports numbers")
				return false
			}

			queryValue, err := strconv.ParseFloat(qf.Value.(string), 64)
			if err != nil {
				logger.WithValues("value", value).Errorf("properties lessthan and greaterthan operator only supports numbers")
				return false
			}

			if qf.Op == grammar.Gt {
				return propertyValue > queryValue
			}

			return propertyValue < queryValue

		default:
			logger.WithValues("operation", qf.Op).Infof("matchGrammar not-implemented")
			return false
		}
	}

	for _, subQf := range qf.Fields {
		// Consider subQf.Operation (AND vs OR)
		match := rs.matchGrammar(subQf, s)
		if !match {
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
