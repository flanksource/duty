package query

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

var (
	allowedColumnFieldsInComponents = []string{"owner", "topology_type"}
)

func GetComponentsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Component, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var components []models.Component
	err := ctx.DB().Where(LocalFilter).Where("id IN ?", ids).Find(&components).Error
	return components, err
}

func FindComponents(ctx context.Context, resourceSelectors types.ResourceSelectors) ([]models.Component, error) {
	items, err := FindComponentIDs(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetComponentsByIDs(ctx, items)
}

func FindComponentIDs(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allComponents []uuid.UUID
	for _, resourceSelector := range resourceSelectors {
		hash := "FindComponents-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			allComponents = append(allComponents, val.([]uuid.UUID)...)
			continue
		}

		if query := resourceSelectorQuery(ctx, resourceSelector, "labels", allowedColumnFieldsInComponents); query != nil {
			var ids []uuid.UUID
			if err := query.Model(&models.Component{}).Find(&ids).Error; err != nil {
				return nil, fmt.Errorf("error getting components with selectors[%v]: %w", resourceSelector, err)
			}

			if len(ids) == 0 {
				cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
			} else {
				cacheToUse.SetDefault(hash, ids)
			}

			allComponents = append(allComponents, ids...)
		}
	}

	return allComponents, nil
}
