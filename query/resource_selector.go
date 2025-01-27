package query

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/duration"
	"github.com/flanksource/commons/logger"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/pkg/kube/labels"
	"github.com/flanksource/duty/query/grammar"
	"github.com/flanksource/duty/types"
)

type SearchResourcesRequest struct {
	// Limit the number of results returned per resource type
	Limit int `json:"limit"`

	Checks     []types.ResourceSelector `json:"checks"`
	Components []types.ResourceSelector `json:"components"`
	Configs    []types.ResourceSelector `json:"configs"`
}

type SearchResourcesResponse struct {
	Checks     []SelectedResource `json:"checks,omitempty"`
	Components []SelectedResource `json:"components,omitempty"`
	Configs    []SelectedResource `json:"configs,omitempty"`
}

func (r *SearchResourcesResponse) GetIDs() []string {
	var ids []string
	ids = append(ids, lo.Map(r.Checks, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Configs, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Components, func(c SelectedResource, _ int) string { return c.ID })...)
	return ids
}

type SelectedResource struct {
	ID        string            `json:"id"`
	Agent     string            `json:"agent"`
	Icon      string            `json:"icon,omitempty"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	Tags      map[string]string `json:"tags,omitempty"`
}

func SearchResources(ctx context.Context, req SearchResourcesRequest) (*SearchResourcesResponse, error) {
	var output SearchResourcesResponse

	if req.Limit <= 0 {
		req.Limit = 100
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if items, err := FindConfigsByResourceSelector(ctx, req.Limit, req.Configs...); err != nil {
			return err
		} else {
			for i := range items {
				output.Configs = append(output.Configs, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     items[i].AgentID.String(),
					Tags:      items[i].Tags,
					Name:      items[i].GetName(),
					Namespace: items[i].GetNamespace(),
					Type:      items[i].GetType(),
				})
			}
		}

		return nil
	})

	eg.Go(func() error {
		if items, err := FindChecks(ctx, req.Limit, req.Checks...); err != nil {
			return err
		} else {
			for i := range items {
				output.Checks = append(output.Checks, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     items[i].AgentID.String(),
					Icon:      items[i].Icon,
					Tags:      items[i].Labels,
					Name:      items[i].GetName(),
					Namespace: items[i].GetNamespace(),
					Type:      items[i].GetType(),
				})
			}
		}

		return nil
	})

	eg.Go(func() error {
		if items, err := FindComponents(ctx, req.Limit, req.Components...); err != nil {
			return err
		} else {
			for i := range items {
				output.Components = append(output.Components, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     items[i].AgentID.String(),
					Icon:      items[i].Icon,
					Tags:      items[i].Labels,
					Name:      items[i].GetName(),
					Namespace: items[i].GetNamespace(),
					Type:      items[i].GetType(),
				})
			}
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return &output, nil
}

func SetResourceSelectorClause(
	ctx context.Context,
	resourceSelector types.ResourceSelector,
	query *gorm.DB,
	table string,
	allowedColumnsAsFields []string,
) (*gorm.DB, error) {
	searchSetAgent := false

	var searchConditions []string
	if resourceSelector.Name != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("name = %q", resourceSelector.Name))
	}

	if resourceSelector.ID != "" {
		searchConditions = append(searchConditions, fmt.Sprintf("id = %q", resourceSelector.ID))
	}

	if len(resourceSelector.Health) != 0 {
		searchConditions = append(searchConditions, fmt.Sprintf("health = %q", resourceSelector.Health))
	}

	for _, resourceType := range resourceSelector.Types {
		searchConditions = append(searchConditions, fmt.Sprintf("type = %q", resourceType))
	}

	for _, resourceType := range resourceSelector.Statuses {
		searchConditions = append(searchConditions, fmt.Sprintf("status = %q", resourceType))
	}

	if len(searchConditions) > 0 {
		joined := strings.Join(searchConditions, " ")
		resourceSelector.Search += fmt.Sprintf(" %s", joined)
	}

	qm, err := GetModelFromTable(table)
	if err != nil {
		return nil, fmt.Errorf("grammar parsing not implemented for table: %s", table)
	}

	if resourceSelector.Search != "" {
		qf, err := grammar.ParsePEG(resourceSelector.Search)
		if err != nil {
			return nil, fmt.Errorf("error parsing grammar[%s]: %w", resourceSelector.Search, err)
		}

		flatFields := grammar.FlatFields(qf)
		if slices.ContainsFunc(flatFields, func(s string) bool { return s == "agent" || s == "agent_id" }) {
			searchSetAgent = true
		}

		var clauses []clause.Expression
		query, clauses, err = qm.Apply(ctx, *qf, query)
		if err != nil {
			return nil, fmt.Errorf("error applying query model: %w", err)
		}

		query = query.Clauses(clauses...)
	}

	if !resourceSelector.IncludeDeleted {
		query = query.Where("deleted_at IS NULL")
	}

	if resourceSelector.Namespace != "" {
		switch table {
		case "config_items":
			query = query.Where("tags->>'namespace' = ?", resourceSelector.Namespace)
		default:
			query = query.Where("namespace = ?", resourceSelector.Namespace)
		}
	}

	var agentID *uuid.UUID
	if !searchSetAgent {
		if !qm.HasAgents {
			return nil, api.Errorf(api.EINVALID, "agent search is not supported for table=%s", table)
		}

		agentID, err := getAgentID(ctx, resourceSelector.Agent)
		if err != nil {
			return nil, err
		}

		if agentID != nil {
			query = query.Where("agent_id = ?", *agentID)
		}
	}

	if resourceSelector.Scope != "" {
		scope, err := getScopeID(ctx, resourceSelector.Scope, table, agentID)
		if err != nil {
			return nil, fmt.Errorf("error fetching scope: %w", err)
		}
		switch table {
		case "checks":
			query = query.Where("canary_id = ?", scope)
		case "config_items":
			query = query.Where("scraper_id = ?", scope)
		case "components":
			query = query.Where("topology_id = ?", scope)
		default:
			return nil, api.Errorf(api.EINVALID, "scope is not supported for %s", table)
		}
	}

	if len(resourceSelector.TagSelector) > 0 {
		if !qm.HasTags {
			return nil, api.Errorf(api.EINVALID, "tagSelector is not supported for table=%s", table)
		} else {
			parsedTagSelector, err := labels.Parse(resourceSelector.TagSelector)
			if err != nil {
				return nil, api.Errorf(api.EINVALID, "failed to parse tag selector: %v", err)
			}
			requirements, _ := parsedTagSelector.Requirements()
			for _, r := range requirements {
				query = tagSelectorRequirementsToSQLClause(query, r)
			}
		}
	}

	if len(resourceSelector.LabelSelector) > 0 {
		if !qm.HasLabels {
			return nil, api.Errorf(api.EINVALID, "labelSelector is not supported for table=%s", table)
		}

		parsedLabelSelector, err := labels.Parse(resourceSelector.LabelSelector)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse label selector: %v", err)
		}
		requirements, _ := parsedLabelSelector.Requirements()
		for _, r := range requirements {
			query = labelSelectorRequirementToSQLClause(query, r)
		}
	}

	if len(resourceSelector.FieldSelector) > 0 {
		parsedFieldSelector, err := labels.Parse(resourceSelector.FieldSelector)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse field selector: %v", err)
		}

		requirements, _ := parsedFieldSelector.Requirements()
		for _, r := range requirements {
			if collections.Contains(allowedColumnsAsFields, r.Key()) {
				query = fieldSelectorRequirementToSQLClause(query, r)
			} else {
				query = propertySelectorRequirementToSQLClause(query, r)
			}
		}
	}

	if resourceSelector.Functions.ComponentConfigTraversal != nil {
		args := resourceSelector.Functions.ComponentConfigTraversal
		if table == "components" {
			query = query.Where("id IN (SELECT id from lookup_component_config_id_related_components(?))", args.ComponentID)
		}
	}

	return query, nil
}

// queryResourceSelector runs the given resourceSelector and returns the resource ids
func queryResourceSelector(
	ctx context.Context,
	limit int,
	resourceSelector types.ResourceSelector,
	table string,
	allowedColumnsAsFields []string,
) ([]uuid.UUID, error) {
	if resourceSelector.IsEmpty() {
		return nil, nil
	}

	hash := fmt.Sprintf("%s-%s-%d", table, resourceSelector.Hash(), limit)

	// NOTE: When RLS is enabled, we need to scope the cache per RLS permission.
	if payload := ctx.RLSPayload(); payload != nil {
		hash += fmt.Sprintf("-rls-%s", payload.Fingerprint())
	}

	cacheToUse := getterCache
	if resourceSelector.Immutable() {
		cacheToUse = immutableCache
	}

	if resourceSelector.Cache != "no-cache" {
		if val, ok := cacheToUse.Get(hash); ok {
			return val.([]uuid.UUID), nil
		}
	}

	query := ctx.DB().Select("id").Table(table)

	// Resource selector's limit gets higher priority
	if resourceSelector.Limit > 0 {
		query = query.Limit(resourceSelector.Limit)
	} else if limit > 0 {
		query = query.Limit(limit)
	}

	query, err := SetResourceSelectorClause(ctx, resourceSelector, query, table, allowedColumnsAsFields)
	if err != nil {
		return nil, err
	}

	var output []uuid.UUID
	if err := query.Find(&output).Error; err != nil {
		return nil, err
	}

	if resourceSelector.Cache != "no-store" {
		cacheDuration := cache.DefaultExpiration
		if len(output) == 0 {
			cacheDuration = time.Minute // if results weren't found, cache it shortly even on the immutable cache
		}

		if strings.HasPrefix(resourceSelector.Cache, "max-age=") {
			d, err := duration.ParseDuration(strings.TrimPrefix(resourceSelector.Cache, "max-age="))
			if err != nil {
				return nil, err
			}

			cacheDuration = time.Duration(d)
		}

		cacheToUse.Set(hash, output, cacheDuration)
	}

	return output, nil
}

// tagSelectorRequirementsToSQLClause to converts each selector requirement into a gorm SQL clause
func tagSelectorRequirementsToSQLClause(q *gorm.DB, r labels.Requirement) *gorm.DB {
	switch r.Operator() {
	case selection.Equals, selection.DoubleEquals:
		for val := range r.Values() {
			q = q.Where("tags @> ?", types.JSONStringMap{r.Key(): val})
		}
	case selection.NotEquals:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("tags->>'%s' != ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
		}
	case selection.In:
		q = q.Where(fmt.Sprintf("tags->>'%s' IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.NotIn:
		q = q.Where(fmt.Sprintf("tags->>'%s' NOT IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.DoesNotExist:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("tags->>'%s' IS NULL", val))
		}
	case selection.Exists:
		q = q.Where("tags ? ?", gorm.Expr("?"), r.Key())
	case selection.GreaterThan:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("tags->>'%s' > ?", r.Key()), val)
		}
	case selection.LessThan:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("tags->>'%s' < ?", r.Key()), val)
		}
	}

	return q
}

// labelSelectorRequirementToSQLClause to converts each selector requirement into a gorm SQL clause
func labelSelectorRequirementToSQLClause(q *gorm.DB, r labels.Requirement) *gorm.DB {
	switch r.Operator() {
	case selection.Equals, selection.DoubleEquals:
		for val := range r.Values() {
			q = q.Where("labels @> ?", types.JSONStringMap{r.Key(): val})
		}
	case selection.NotEquals:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("labels->>'%s' != ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
		}
	case selection.In:
		q = q.Where(fmt.Sprintf("labels->>'%s' IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.NotIn:
		q = q.Where(fmt.Sprintf("labels->>'%s' NOT IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.DoesNotExist:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("labels->>'%s' IS NULL", val))
		}
	case selection.Exists:
		q = q.Where("labels ? ?", gorm.Expr("?"), r.Key())
	case selection.GreaterThan:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("labels->>'%s' > ?", r.Key()), val)
		}
	case selection.LessThan:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("labels->>'%s' < ?", r.Key()), val)
		}
	}

	return q
}

// fieldSelectorRequirementToSQLClause to converts each selector requirement into a gorm SQL clause
func fieldSelectorRequirementToSQLClause(q *gorm.DB, r labels.Requirement) *gorm.DB {
	switch r.Operator() {
	case selection.Equals, selection.DoubleEquals:
		for val := range r.Values() {
			if r.Key() == "external_id" {
				q = q.Where(fmt.Sprintf("? = ANY(%s)", r.Key()), lo.Ternary[any](val == "nil", nil, val))
			} else {
				q = q.Where(fmt.Sprintf("%s = ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
			}
		}
	case selection.NotEquals:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("%s <> ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
		}
	case selection.In:
		q = q.Where(fmt.Sprintf("%s IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.NotIn:
		q = q.Where(fmt.Sprintf("%s NOT IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.GreaterThan:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("%s > ?", r.Key()), val)
		}
	case selection.LessThan:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("%s < ?", r.Key()), val)
		}
	case selection.Exists, selection.DoesNotExist:
		logger.Warnf("Operators %s is not supported for property lookup", r.Operator())
	}

	return q
}

// propertySelectorRequirementToSQLClause to converts each selector requirement into a gorm SQL clause
func propertySelectorRequirementToSQLClause(q *gorm.DB, r labels.Requirement) *gorm.DB {
	switch r.Operator() {
	case selection.Equals, selection.DoubleEquals:
		for val := range r.Values() {
			q = q.Where("properties @> ?", types.Properties{{Name: r.Key(), Text: val}})
		}
	case selection.NotEquals:
		for val := range r.Values() {
			q = q.Where("NOT (properties @> ?)", types.Properties{{Name: r.Key(), Text: val}})
		}
	case selection.GreaterThan,
		selection.LessThan,
		selection.In,
		selection.NotIn,
		selection.Exists,
		selection.DoesNotExist:
		logger.Warnf("TODO: Implement %s for property lookup", r.Operator())
	}

	return q
}

// getScopeID takes either uuid or namespace/name and table to return the appropriate scope_id
func getScopeID(ctx context.Context, scope string, table string, agentID *uuid.UUID) (string, error) {
	// If scope is a valid uuid, return as is
	if _, err := uuid.Parse(scope); err == nil {
		return scope, nil
	}

	key := table + scope
	if cacheVal, exists := scopeCache.Get(key); exists {
		return cacheVal.(string), nil
	}

	parts := strings.Split(scope, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("scope should be either uuid or namespace/name format")
	}
	namespace, name := parts[0], parts[1]

	q := ctx.DB()
	switch table {
	case "checks":
		q = q.Table("canaries").Select("id").Where("name = ? AND namespace = ?", name, namespace)
	case "config_items":
		q = q.Table("config_scrapers").Select("id").Where("name = ?", namespace+"/"+name)
	case "components":
		q = q.Table("topologies").Select("id").Where("name = ? AND namespace = ?", name, namespace)
	default:
		return "", api.Errorf(api.EINVALID, "scope is not supported for %s", table)
	}

	if agentID != nil {
		q = q.Where("agent_id = ?", *agentID)
	}

	var id string
	tx := q.Find(&id)
	if tx.RowsAffected != 1 {
		agentToLog := "all"
		if agentID != nil {
			agentToLog = agentID.String()
		}
		ctx.Warnf(
			"multiple agents returned for resource selector with [scope=%s, table=%s, agent=%s]",
			scope,
			table,
			agentToLog,
		)
	}
	if tx.Error != nil {
		return "", tx.Error
	}

	scopeCache.Set(key, id, cache.NoExpiration)
	return id, nil
}

func getAgentID(ctx context.Context, agent string) (*uuid.UUID, error) {
	if agent == "" {
		return &uuid.Nil, nil
	}
	if agent == "all" {
		return nil, nil
	}

	if uid, err := uuid.Parse(agent); err == nil {
		return &uid, nil
	}

	agentModel, err := FindCachedAgent(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("could not find agent[%s]: %w", agent, err)
	}
	return &agentModel.ID, nil
}

func queryTableWithResourceSelectors(
	ctx context.Context,
	table string,
	allowedFields []string,
	limit int,
	resourceSelectors ...types.ResourceSelector,
) ([]uuid.UUID, error) {
	var output []uuid.UUID

	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector(ctx, limit, resourceSelector, table, allowedFields)
		if err != nil {
			return nil, err
		}

		output = append(output, items...)
		if limit > 0 && len(output) >= limit {
			return output[:limit], nil
		}
	}

	return output, nil
}
