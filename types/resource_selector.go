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
	"github.com/flanksource/is-healthy/pkg/health"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/flanksource/duty/query/grammar"
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

func IsMatchItem(q string) bool {
	return strings.ContainsAny(q, "*!,")
}

// ToListOptions converts the resource selector to a ListOptions, using the supported optiions by Kubernetes List, it returns true if the query can be executed entirely by Kubernetes
func (c ResourceSelector) ToListOptions() (metav1.ListOptions, bool) {
	opts := metav1.ListOptions{
		LabelSelector: c.LabelSelector,
		FieldSelector: c.FieldSelector,
	}

	if c.Search != "" || IsMatchItem(c.Name) {
		return opts, false
	}
	return opts, true
}

func (c ResourceSelector) ToGetOptions() (string, bool) {
	name := c.Name

	if name != "" && c.Search == "" && !IsMatchItem(name) && (c.Namespace != "" || c.Types.Contains("namespace")) {
		return name, true
	}

	return "", false
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

func quote(s string) string {
	//FIXME only quote if needed, remove quotes if unnecessary
	return s
}
func (rs ResourceSelector) ToPeg(convertSelectors bool) string {
	var searchConditions []string

	if rs.ID != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("id=%q", quote(rs.ID)))
	}

	if rs.Name != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("name=%q", quote(rs.Name)))
	}

	if rs.Namespace != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("namespace=%q", quote(rs.Namespace)))
	}

	if len(rs.Health) != 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("health=%q", quote(string(rs.Health))))
	}

	if len(rs.Types) > 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("type=%q", strings.Join(rs.Types, ",")))
	}

	if len(rs.Statuses) > 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("status=%q", strings.Join(rs.Statuses, ",")))
	}

	if rs.Agent != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("agent=%q", quote(rs.Agent)))
	}

	if convertSelectors {
		// Adding this flag for now until we migrate matchItems support in the SQL query
		if rs.LabelSelector != "" {
			searchConditions = append(searchConditions, selectorToPegCondition("labels.", rs.LabelSelector)...)
		}

		if rs.TagSelector != "" {
			searchConditions = append(searchConditions, selectorToPegCondition("tags.", rs.TagSelector)...)
		}

		if rs.FieldSelector != "" {
			searchConditions = append(searchConditions, selectorToPegCondition("", rs.FieldSelector)...)
		}
	}

	peg := rs.Search
	if len(searchConditions) > 0 {
		joined := strings.Join(searchConditions, " ")
		peg += fmt.Sprintf(" %s", joined)
	}

	return peg
}

func (rs ResourceSelector) Type(t string) ResourceSelector {
	rs.Types = append(rs.Types, t)
	return rs
}

func (rs ResourceSelector) MetadataOnly() ResourceSelector {
	rs.Cache = "metadata"
	return rs
}

func (rs ResourceSelector) IsMetadataOnly() bool {
	return rs.Cache == "metadata"
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

	peg := rs.ToPeg(true)
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

		value, err := extractResourceFieldValue(s, qf.Field)
		if err != nil {
			logger.Errorf("failed to extract value for field: %v", qf.Field)
			return false
		}

		var patterns []string
		if qfs, ok := qf.Value.(string); ok {
			patterns = strings.Split(qfs, ",")
		}

		switch qf.Op {
		case grammar.Eq:
			// Special case: agent=all should match any agent
			// This mirrors the behavior in query.SetResourceSelectorClause where
			// agent=all results in no agent filter being applied
			if qf.Field == "agent" && len(patterns) == 1 && patterns[0] == "all" {
				return true
			}
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

	var matchAny bool
	for _, subQf := range qf.Fields {
		match := rs.matchGrammar(subQf, s)
		if match {
			matchAny = true
		}

		if qf.Op == "and" && !match {
			return false // fail early
		}
	}

	return matchAny
}

func (rs ResourceSelector) String() string {
	return strings.Trim(rs.ToPeg(true), " ")
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

func (rs ResourceSelectors) Matches(s ResourceSelectable) bool {
	if len(rs) == 0 {
		return true // an empty selector matches everything
	}

	for _, selector := range rs {
		if selector.Matches(s) {
			return true
		}
	}
	return false
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

type DescriptionProvider interface {
	GetHealthDescription() string
}

type AgentProvider interface {
	GetAgentID() string
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

func extractResourceFieldValue(rs ResourceSelectable, field string) (string, error) {
	switch field {
	case "name":
		return rs.GetName(), nil
	case "namespace":
		return rs.GetNamespace(), nil
	case "id":
		return rs.GetID(), nil
	case "type":
		return rs.GetType(), nil
	case "status":
		value, err := rs.GetStatus()
		if err != nil {
			return "", fmt.Errorf("failed to get status: %w", err)
		}
		return value, nil
	case "health":
		value, err := rs.GetHealth()
		if err != nil {
			return "", fmt.Errorf("failed to get health: %w", err)
		}
		return value, nil
	case "agent":
		if agentProvider, ok := rs.(AgentProvider); ok {
			return agentProvider.GetAgentID(), nil
		}
		return "", nil
	}

	if strings.HasPrefix(field, "labels.") {
		key := strings.TrimSpace(strings.TrimPrefix(field, "labels."))
		return rs.GetLabelsMatcher().Get(key), nil
	} else if strings.HasPrefix(field, "tags.") {
		key := strings.TrimSpace(strings.TrimPrefix(field, "tags."))
		if tagsMatcher, ok := rs.(TagsMatchable); ok && tagsMatcher.GetTagsMatcher() != nil {
			return tagsMatcher.GetTagsMatcher().Get(key), nil
		}
	} else if strings.HasPrefix(field, "properties.") {
		propertyName := strings.TrimSpace(strings.TrimPrefix(field, "properties."))
		propertiesJSON := rs.GetFieldsMatcher().Get("properties")
		var properties Properties
		if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
			return "", fmt.Errorf("failed to unmarshall properties: %w", err)
		}

		for _, p := range properties {
			if p.Name != propertyName {
				continue
			}

			if p.Text != "" {
				return p.Text, nil
			} else if p.Value != nil {
				return strconv.FormatInt(*p.Value, 10), nil
			}
		}
	}

	// Unknown key is a field selector
	return rs.GetFieldsMatcher().Get(strings.TrimSpace(field)), nil
}

var _ ResourceSelectable = ResourceSelectableMap{}

type ResourceSelectableMap map[string]any

func (t ResourceSelectableMap) GetFieldsMatcher() fields.Fields {
	return GenericFieldMatcher{t}
}

func (r ResourceSelectableMap) GetLabelsMatcher() labels.Labels {
	labelsRaw, ok := r["labels"]
	if !ok {
		return nil
	}

	if labels, ok := labelsRaw.(map[string]string); ok {
		return GenericLabelsMatcher{labels}
	}

	if labels, ok := labelsRaw.(map[string]any); ok {
		return GenericLabelsMatcherAny{labels}
	}

	return nil
}

func (r ResourceSelectableMap) GetTagsMatcher() labels.Labels {
	tagsRaw, ok := r["tags"]
	if !ok {
		return nil
	}

	if labels, ok := tagsRaw.(map[string]string); ok {
		return GenericLabelsMatcher{labels}
	}

	if labels, ok := r["tags"].(map[string]any); ok {
		return GenericLabelsMatcherAny{labels}
	}

	return nil
}

func (t ResourceSelectableMap) GetID() string {
	return t["id"].(string)
}

func (t ResourceSelectableMap) GetName() string {
	return t["name"].(string)
}

func (t ResourceSelectableMap) GetNamespace() string {
	if ns, ok := t["namespace"].(string); ok && ns != "" {
		return ns
	}

	if tags, ok := t["tags"].(map[string]string); ok {
		return tags["namespace"]
	}

	return ""
}

func (t ResourceSelectableMap) GetType() string {
	itemType, ok := t["type"].(string)
	if !ok {
		return ""
	}

	return itemType
}

func (t ResourceSelectableMap) GetHealthDescription() string {
	healthDescription, ok := t["description"].(string)
	if !ok {
		return ""
	}

	return healthDescription
}

func (t ResourceSelectableMap) GetStatus() (string, error) {
	status, ok := t["status"].(string)
	if !ok {
		return "", nil
	}

	return status, nil
}

func (t ResourceSelectableMap) GetHealth() (string, error) {
	health, ok := t["health"].(string)
	if !ok {
		return "", nil
	}

	return health, nil
}

type GenericFieldMatcher struct {
	Fields map[string]any
}

func (c GenericFieldMatcher) Get(key string) string {
	val := c.Fields[key]
	switch v := val.(type) {
	case string:
		return v
	default:
		marshalled, _ := json.Marshal(v)
		return string(marshalled)
	}
}

func (c GenericFieldMatcher) Has(key string) bool {
	_, ok := c.Fields[key]
	return ok
}

func (c GenericFieldMatcher) Lookup(key string) (value string, exists bool) {
	val, exists := c.Fields[key]
	if !exists {
		return "", false
	}

	switch v := val.(type) {
	case string:
		return v, true
	default:
		marshalled, _ := json.Marshal(v)
		return string(marshalled), true
	}
}

type GenericLabelsMatcher struct {
	Map map[string]string
}

func (c GenericLabelsMatcher) Get(key string) string {
	return c.Map[key]
}

func (c GenericLabelsMatcher) Has(key string) bool {
	_, ok := c.Map[key]
	return ok
}

// Lookup returns the value for the provided label if it exists and whether the provided label exist
func (c GenericLabelsMatcher) Lookup(label string) (value string, exists bool) {
	value, exists = c.Map[label]
	return
}

type GenericLabelsMatcherAny struct {
	Map map[string]any
}

func (c GenericLabelsMatcherAny) Get(key string) string {
	return fmt.Sprintf("%v", c.Map[key])
}

func (c GenericLabelsMatcherAny) Has(key string) bool {
	_, ok := c.Map[key]
	return ok
}

func (c GenericLabelsMatcherAny) Lookup(label string) (value string, exists bool) {
	val, exists := c.Map[label]
	if !exists {
		return "", false
	}
	return fmt.Sprintf("%v", val), true
}

type UnstructuredResource struct {
	*unstructured.Unstructured
}

func (u *UnstructuredResource) GetFieldsMatcher() fields.Fields {
	return GenericFieldMatcher{Fields: u.Object}
}

func (u *UnstructuredResource) GetLabelsMatcher() labels.Labels {
	return labels.Set(u.GetLabels())
}

func (u *UnstructuredResource) GetID() string {
	return string(u.Unstructured.GetUID())
}

func (u *UnstructuredResource) GetName() string {
	return u.Unstructured.GetName()
}

func (u *UnstructuredResource) GetNamespace() string {
	return u.Unstructured.GetNamespace()
}

func (u *UnstructuredResource) GetType() string {
	return u.GetKind()
}

func (u *UnstructuredResource) GetStatus() (string, error) {
	healthStatus, err := health.GetDefaultHealth(u.Unstructured)
	if err != nil {
		return "", err
	}
	return string(healthStatus.Status), nil
}

func (u UnstructuredResource) GetHealth() (string, error) {
	healthStatus, err := health.GetDefaultHealth(u.Unstructured)
	if err != nil {
		return "", err
	}
	return string(healthStatus.Health), nil
}
