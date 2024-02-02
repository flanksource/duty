package query

import (
	"fmt"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// resourceSelectorQuery returns an ANDed query from all the fields
func resourceSelectorQuery(ctx context.Context, resourceSelector types.ResourceSelector, labelsColumn string, allowedColumnsAsFields []string) *gorm.DB {
	if resourceSelector.IsEmpty() {
		return nil
	}

	query := ctx.DB().Debug().Select("id").Where("deleted_at IS NULL")

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

	if resourceSelector.Agent != "" {
		if resourceSelector.Agent == "self" {
			query = query.Where("agent_id = ?", uuid.Nil)
		} else if uid, err := uuid.Parse(resourceSelector.Agent); err == nil {
			query = query.Where("agent_id = ?", uid)
		} else { // assume it's an agent name
			agent, err := FindCachedAgent(ctx, resourceSelector.Agent)
			if err != nil {
				return nil
			}
			query = query.Where("agent_id = ?", agent.ID)
		}
	}

	if len(resourceSelector.LabelSelector) > 0 {
		labels := collections.SelectorToMap(resourceSelector.LabelSelector)
		var onlyKeys []string
		for k, v := range labels {
			if v == "" {
				onlyKeys = append(onlyKeys, k)
				delete(labels, k)
			}
		}

		query = query.Where(fmt.Sprintf("%s @> ?", labelsColumn), types.JSONStringMap(labels))
		for _, k := range onlyKeys {
			query = query.Where(fmt.Sprintf("%s ?? ?", labelsColumn), k)
		}
	}

	if len(resourceSelector.FieldSelector) > 0 {
		fields := collections.SelectorToMap(resourceSelector.FieldSelector)
		columnWhereClauses := map[string]string{}
		var props models.Properties
		for k, v := range fields {
			if collections.Contains(allowedColumnsAsFields, k) {
				columnWhereClauses[k] = v
			} else {
				props = append(props, &models.Property{Name: k, Text: v})
			}
		}

		if len(columnWhereClauses) > 0 {
			query = query.Where(columnWhereClauses)
		}

		if len(props) > 0 {
			query = query.Where("properties @> ?", props)
		}
	}

	return query
}
