package query

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/duration"
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

	Canaries      []types.ResourceSelector `json:"canaries"`
	Checks        []types.ResourceSelector `json:"checks"`
	Components    []types.ResourceSelector `json:"components"`
	Configs       []types.ResourceSelector `json:"configs"`
	ConfigChanges []types.ResourceSelector `json:"config_changes"`
	Playbooks     []types.ResourceSelector `json:"playbooks"`
	Connections   []types.ResourceSelector `json:"connections"`
}

type SearchResourcesResponse struct {
	Canaries      []SelectedResource `json:"canaries,omitempty"`
	Checks        []SelectedResource `json:"checks,omitempty"`
	Components    []SelectedResource `json:"components,omitempty"`
	Configs       []SelectedResource `json:"configs,omitempty"`
	ConfigChanges []SelectedResource `json:"config_changes,omitempty"`
	Playbooks     []SelectedResource `json:"playbooks,omitempty"`
	Connections   []SelectedResource `json:"connections,omitempty"`
}

func (r *SearchResourcesResponse) GetIDs() []string {
	var ids []string
	ids = append(ids, lo.Map(r.Canaries, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Checks, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Configs, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Components, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.ConfigChanges, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Playbooks, func(c SelectedResource, _ int) string { return c.ID })...)
	ids = append(ids, lo.Map(r.Connections, func(c SelectedResource, _ int) string { return c.ID })...)
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
		if items, err := FindCanaries(ctx, req.Limit, req.Canaries...); err != nil {
			return err
		} else {
			for i := range items {
				output.Canaries = append(output.Canaries, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     items[i].AgentID.String(),
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

	eg.Go(func() error {
		if items, err := FindConfigChangesByResourceSelector(ctx, req.Limit, req.ConfigChanges...); err != nil {
			return err
		} else {
			for i := range items {
				agentID := ""
				if items[i].AgentID != nil {
					agentID = items[i].AgentID.String()
				}
				output.ConfigChanges = append(output.ConfigChanges, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     agentID,
					Name:      items[i].GetName(),
					Namespace: items[i].GetNamespace(),
					Type:      items[i].GetType(),
				})
			}
		}

		return nil
	})

	eg.Go(func() error {
		if items, err := FindPlaybooksByResourceSelector(ctx, req.Limit, req.Playbooks...); err != nil {
			return err
		} else {
			for i := range items {
				output.Playbooks = append(output.Playbooks, SelectedResource{
					ID:        items[i].GetID(),
					Name:      items[i].GetName(),
					Namespace: items[i].GetNamespace(),
					Type:      items[i].GetType(),
				})
			}
		}

		return nil
	})

	eg.Go(func() error {
		if items, err := FindConnectionsByResourceSelector(ctx, req.Limit, req.Connections...); err != nil {
			return err
		} else {
			for i := range items {
				output.Connections = append(output.Connections, SelectedResource{
					ID:        items[i].GetID(),
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
) (*gorm.DB, error) {
	searchSetAgent := false

	qm, err := GetModelFromTable(table)
	if err != nil {
		return nil, fmt.Errorf("grammar parsing not implemented for table: %s", table)
	}

	if peg := resourceSelector.ToPeg(false); peg != "" {
		qf, err := grammar.ParsePEG(peg)
		if err != nil {
			return nil, fmt.Errorf("error parsing grammar[%s]: %w", peg, err)
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

	var agentID *uuid.UUID
	if !searchSetAgent && qm.HasAgents {
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
		case "config_changes", "catalog_changes":
			query = query.Where("config_id = ?", scope)
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
				query = jsonColumnRequirementsToSQLClause(query, "tags", r)
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
			query = jsonColumnRequirementsToSQLClause(query, "labels", r)
		}
	}

	if len(resourceSelector.FieldSelector) > 0 {
		parsedFieldSelector, err := labels.Parse(resourceSelector.FieldSelector)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse field selector: %v", err)
		}

		requirements, _ := parsedFieldSelector.Requirements()
		for _, r := range requirements {
			query = jsonColumnRequirementsToSQLClause(query, "properties", r)
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
func queryResourceSelector[T any](
	ctx context.Context,
	limit int,
	selectColumns []string,
	resourceSelector types.ResourceSelector,
	table string,
	clauses ...clause.Expression,
) ([]T, error) {
	if resourceSelector.IsEmpty() {
		return nil, nil
	}

	// must create a deep copy to avoid mutating the original order of the select columns
	var selectColumnsCopy = make([]string, len(selectColumns))
	copy(selectColumnsCopy, selectColumns)
	sort.Strings(selectColumnsCopy)

	var dummy T
	cacheKey := fmt.Sprintf("%s-%s-%s-%d-%T", strings.Join(selectColumnsCopy, ","), table, resourceSelector.Hash(), limit, dummy)

	// NOTE: When RLS is enabled, we need to scope the cache per RLS permission.
	if payload := ctx.RLSPayload(); payload != nil {
		cacheKey += fmt.Sprintf("-rls-%s", payload.Fingerprint())
	}

	cacheToUse := getterCache
	if resourceSelector.Immutable() {
		cacheToUse = immutableCache
	}

	if resourceSelector.Cache != "no-cache" {
		if val, ok := cacheToUse.Get(cacheKey); ok {
			return val.([]T), nil
		}
	}

	query := ctx.DB().Select(selectColumns).Table(table)
	if len(clauses) > 0 {
		query = query.Clauses(clauses...)
	}

	// Resource selector's limit gets higher priority
	if resourceSelector.Limit > 0 {
		query = query.Limit(resourceSelector.Limit)
	} else if limit > 0 {
		query = query.Limit(limit)
	}

	query, err := SetResourceSelectorClause(ctx, resourceSelector, query, table)
	if err != nil {
		return nil, err
	}

	if ctx.Properties().String("log.level.resourceSelector", "") != "" {
		ctx.WithName("resourceSelector").Logger.WithValues("cacheKey", cacheKey).Tracef("query: %s", query.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Find(&[]T{})
		}))
	}

	var output []T
	if err := query.Find(&output).Error; err != nil {
		return nil, err
	}

	if resourceSelector.Cache != "no-store" {
		cacheDuration := cache.DefaultExpiration
		if len(output) == 0 {
			cacheDuration = time.Minute // if results weren't found, cache it shortly even on the immutable cache
		}

		if after, ok := strings.CutPrefix(resourceSelector.Cache, "max-age="); ok {
			d, err := duration.ParseDuration(after)
			if err != nil {
				return nil, err
			}

			cacheDuration = time.Duration(d)
		}

		cacheToUse.Set(cacheKey, output, cacheDuration)
	}

	return output, nil
}

// jsonColumnRequirementsToGormClause converts a selector requirement into gorm clause expressions for a JSON column
func jsonColumnRequirementsToGormClause(column string, r labels.Requirement) []clause.Expression {
	var clauses []clause.Expression

	switch r.Operator() {
	case selection.Equals, selection.DoubleEquals:
		for val := range r.Values() {
			clauses = append(clauses, clause.Expr{
				SQL:  fmt.Sprintf("%s @> ?", column),
				Vars: []any{types.JSONStringMap{r.Key(): val}},
			})
		}
	case selection.NotEquals:
		for val := range r.Values() {
			clauses = append(clauses, clause.Expr{
				SQL:  fmt.Sprintf("%s->>'%s' != ?", column, r.Key()),
				Vars: []any{lo.Ternary[any](val == "nil", nil, val)},
			})
		}
	case selection.In:
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf("%s->>'%s' IN ?", column, r.Key()),
			Vars: []any{collections.MapKeys(r.Values())},
		})
	case selection.NotIn:
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf("%s->>'%s' NOT IN ?", column, r.Key()),
			Vars: []any{collections.MapKeys(r.Values())},
		})
	case selection.DoesNotExist:
		clauses = append(clauses, clause.Expr{
			SQL: fmt.Sprintf("%s->>'%s' IS NULL", column, r.Key()),
		})
	case selection.Exists:
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf("%s ? ?", column),
			Vars: []any{gorm.Expr("?"), r.Key()},
		})
	case selection.GreaterThan:
		for val := range r.Values() {
			clauses = append(clauses, clause.Expr{
				SQL:  fmt.Sprintf("%s->>'%s' > ?", column, r.Key()),
				Vars: []any{val},
			})
		}
	case selection.LessThan:
		for val := range r.Values() {
			clauses = append(clauses, clause.Expr{
				SQL:  fmt.Sprintf("%s->>'%s' < ?", column, r.Key()),
				Vars: []any{val},
			})
		}
	}

	return clauses
}

// jsonColumnRequirementsToSQLClause converts each selector requirement into a gorm SQL clause for a column
func jsonColumnRequirementsToSQLClause(q *gorm.DB, column string, r labels.Requirement) *gorm.DB {
	for _, c := range jsonColumnRequirementsToGormClause(column, r) {
		q = q.Clauses(c)
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
	case "config_changes":
		q = q.Table("config_items").Select("id").Where("name = ? AND namespace = ?", name, namespace)
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
	limit int,
	resourceSelectors ...types.ResourceSelector,
) ([]uuid.UUID, error) {
	var output []uuid.UUID

	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector[uuid.UUID](ctx, limit, []string{"id"}, resourceSelector, table)
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

func QueryTableColumnsWithResourceSelectors[T any](
	ctx context.Context,
	table string,
	selectColumns []string,
	limit int,
	clauses []clause.Expression,
	resourceSelectors ...types.ResourceSelector,
) ([]T, error) {
	var output []T

	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector[T](ctx, limit, selectColumns, resourceSelector, table, clauses...)
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
