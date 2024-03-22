package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/duration"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

type SearchResourcesRequest struct {
	Checks     []types.ResourceSelector `json:"checks"`
	Components []types.ResourceSelector `json:"components"`
	Configs    []types.ResourceSelector `json:"configs"`
}

type SelectedResourceType string

const (
	SelectedResourceTypeCheck     SelectedResourceType = "check"
	SelectedResourceTypeComponent SelectedResourceType = "component"
	SelectedResourceTypeConfig    SelectedResourceType = "config"
)

type SelectedResources struct {
	ID   string               `json:"id"`
	Icon string               `json:"icon"`
	Name string               `json:"name"`
	Type SelectedResourceType `json:"type"`
}

func SearchResources(ctx context.Context, req SearchResourcesRequest) ([]SelectedResources, error) {
	var output []SelectedResources

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if items, err := FindConfigsByResourceSelector(ctx, req.Configs...); err != nil {
			return err
		} else {
			for i := range items {
				output = append(output, SelectedResources{
					ID:   items[i].GetID(),
					Name: items[i].GetName(),
					Type: SelectedResourceTypeConfig,
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
				output = append(output, SelectedResources{
					ID:   items[i].ID.String(),
					Name: items[i].Name,
					Icon: items[i].Icon,
					Type: SelectedResourceTypeCheck,
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
				output = append(output, SelectedResources{
					ID:   items[i].ID.String(),
					Name: items[i].Name,
					Icon: items[i].Icon,
					Type: SelectedResourceTypeComponent,
				})
			}
		}

		return nil
	})

	return output, eg.Wait()
}

// queryResourceSelector runs the given resourceSelector and returns the resource ids
func queryResourceSelector(ctx context.Context, resourceSelector types.ResourceSelector, table, labelsColumn string, allowedColumnsAsFields []string) ([]uuid.UUID, error) {
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
		query = query.Where("namespace = ?", resourceSelector.Namespace)
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

	if len(resourceSelector.LabelSelector) > 0 {
		// parsedLabelSelector, err := labels.Parse(resourceSelector.LabelSelector)
		// if err != nil {
		// 	return nil, api.Errorf(api.EINVALID, fmt.Sprintf("failed to parse label selector: %v", err))
		// }
		// requirements, selectable := parsedLabelSelector.Requirements()
		// if selectable {
		// 	for _, r := range requirements {
		// 		var onlyKeys []string
		// 		onlyKeys = append(onlyKeys, k)
		// 		delete(labelsMap, k)
		// 	}
		//
		// 	query = query.Where(fmt.Sprintf("%s @> ?", labelsColumn), types.JSONStringMap(labelsMap))
		// 	for _, k := range onlyKeys {
		// 		query = query.Where(fmt.Sprintf("%s ? ?", labelsColumn), gorm.Expr("?"), k)
		// 	}
		// }
	}

	if len(resourceSelector.FieldSelector) > 0 {
		parsedFieldSelector, err := labels.Parse(resourceSelector.FieldSelector)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, fmt.Sprintf("failed to parse field selector: %v", err))
		}

		requirements, _ := parsedFieldSelector.Requirements()
		var props models.Properties
		for _, r := range requirements {
			if collections.Contains(allowedColumnsAsFields, r.Key()) {
				query = requirementToSQLClause(query, r)
			} else {
				for v := range r.Values() {
					props = append(props, &models.Property{Name: r.Key(), Text: v})
				}
			}
		}

		if len(props) > 0 {
			query = query.Where("properties @> ?", props)
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

// requirementToSQLClause to converts each selector requirement into a gorm SQL clause
func requirementToSQLClause(q *gorm.DB, r labels.Requirement) *gorm.DB {
	switch r.Operator() {
	case selection.Equals, selection.DoubleEquals:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("%s = ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
		}
	case selection.NotEquals:
		for val := range r.Values() {
			q = q.Where(fmt.Sprintf("%s <> ?", r.Key()), lo.Ternary[any](val == "nil", nil, val))
		}
	case selection.DoesNotExist:
	case selection.In:
		q = q.Where(fmt.Sprintf("%s IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.NotIn:
		q = q.Where(fmt.Sprintf("%s NOT IN ?", r.Key()), collections.MapKeys(r.Values()))
	case selection.Exists:
	case selection.GreaterThan:
	case selection.LessThan:
	}

	return q
}
