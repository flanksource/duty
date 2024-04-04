package query

import (
	"fmt"
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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

type SearchResourcesRequest struct {
	Checks     []types.ResourceSelector `json:"checks"`
	Components []types.ResourceSelector `json:"components"`
	Configs    []types.ResourceSelector `json:"configs"`
}

type SearchResourcesResponse struct {
	Checks     []SelectedResource `json:"checks,omitempty"`
	Components []SelectedResource `json:"components,omitempty"`
	Configs    []SelectedResource `json:"configs,omitempty"`
}

type SelectedResource struct {
	ID        string `json:"id"`
	Agent     string `json:"agent"`
	Icon      string `json:"icon,omitempty"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`

	// Components & checks return labels
	Labels map[string]string `json:"labels,omitempty"`

	// Configs return tags
	Tags map[string]string `json:"tags,omitempty"`
}

func SearchResources(ctx context.Context, req SearchResourcesRequest) (*SearchResourcesResponse, error) {
	var output SearchResourcesResponse

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if items, err := FindConfigsByResourceSelector(ctx, req.Configs...); err != nil {
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
		if items, err := FindChecks(ctx, req.Checks...); err != nil {
			return err
		} else {
			for i := range items {
				output.Checks = append(output.Checks, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     items[i].AgentID.String(),
					Icon:      items[i].Icon,
					Labels:    items[i].Labels,
					Name:      items[i].GetName(),
					Namespace: items[i].GetNamespace(),
					Type:      items[i].GetType(),
				})
			}
		}

		return nil
	})

	eg.Go(func() error {
		if items, err := FindComponents(ctx, req.Components...); err != nil {
			return err
		} else {
			for i := range items {
				output.Components = append(output.Components, SelectedResource{
					ID:        items[i].GetID(),
					Agent:     items[i].AgentID.String(),
					Icon:      items[i].Icon,
					Labels:    items[i].Labels,
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

// queryResourceSelector runs the given resourceSelector and returns the resource ids
func queryResourceSelector(ctx context.Context, resourceSelector types.ResourceSelector, table string, allowedColumnsAsFields []string) ([]uuid.UUID, error) {
	if resourceSelector.IsEmpty() {
		return nil, nil
	}

	hash := fmt.Sprintf("%s-%s", table, resourceSelector.Hash())
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

	if !resourceSelector.IncludeDeleted {
		query = query.Where("deleted_at IS NULL")
	}

	if resourceSelector.ID != "" {
		query = query.Where("id = ?", resourceSelector.ID)
	}
	if resourceSelector.Name != "" {
		query = query.Where("name = ?", resourceSelector.Name)
	}
	if resourceSelector.Namespace != "" {
		switch table {
		case "config_items":
			query = query.Where("tags->>'namespace' = ?", resourceSelector.Namespace)
		default:
			query = query.Where("namespace = ?", resourceSelector.Namespace)
		}
	}
	if len(resourceSelector.Types) != 0 {
		query = query.Where("type IN ?", resourceSelector.Types)
	}
	if len(resourceSelector.Statuses) != 0 {
		query = query.Where("status IN ?", resourceSelector.Statuses)
	}

	if resourceSelector.Agent == "" {
		query = query.Where("agent_id = ?", uuid.Nil)
	} else if resourceSelector.Agent == "all" {
		// do nothing
	} else if uid, err := uuid.Parse(resourceSelector.Agent); err == nil {
		query = query.Where("agent_id = ?", uid)
	} else { // assume it's an agent name
		agent, err := FindCachedAgent(ctx, resourceSelector.Agent)
		if err != nil {
			return nil, err
		}
		query = query.Where("agent_id = ?", agent.ID)
	}

	if len(resourceSelector.TagSelector) > 0 {
		if table != "config_items" {
			return nil, api.Errorf(api.EINVALID, "tag selector is only supported for config_items")
		} else {
			parsedTagSelector, err := labels.Parse(resourceSelector.TagSelector)
			if err != nil {
				return nil, api.Errorf(api.EINVALID, fmt.Sprintf("failed to parse tag selector: %v", err))
			}
			requirements, _ := parsedTagSelector.Requirements()
			for _, r := range requirements {
				query = tagSelectorRequirementsToSQLClause(query, r)
			}
		}
	}

	if len(resourceSelector.LabelSelector) > 0 {
		parsedLabelSelector, err := labels.Parse(resourceSelector.LabelSelector)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, fmt.Sprintf("failed to parse label selector: %v", err))
		}
		requirements, _ := parsedLabelSelector.Requirements()
		for _, r := range requirements {
			query = labelSelectorRequirementToSQLClause(query, r)
		}
	}

	if len(resourceSelector.FieldSelector) > 0 {
		parsedFieldSelector, err := labels.Parse(resourceSelector.FieldSelector)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, fmt.Sprintf("failed to parse field selector: %v", err))
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
			q = q.Where(fmt.Sprintf("%s = ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
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
	case selection.GreaterThan, selection.LessThan, selection.In, selection.NotIn, selection.Exists, selection.DoesNotExist:
		logger.Warnf("TODO: Implement %s for property lookup", r.Operator())
	}

	return q
}
