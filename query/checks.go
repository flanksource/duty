package query

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func FindChecks(ctx context.Context, resourceSelectors types.ResourceSelectors) ([]models.Check, error) {
	ids, err := FindCheckIDs(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetChecksByIDs(ctx, ids)
}

func FindCheckIDs(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	for _, rs := range resourceSelectors {
		if rs.FieldSelector != "" {
			return nil, fmt.Errorf("field selector is not supported for checks (%s)", rs.FieldSelector)
		}
	}

	var allChecks []uuid.UUID
	for _, resourceSelector := range resourceSelectors {
		hash := "FindChecks-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			allChecks = append(allChecks, val.([]uuid.UUID)...)
			continue
		}

		if query := resourceSelectorQuery(ctx, resourceSelector, "labels", nil); query != nil {
			var ids []uuid.UUID
			if err := query.Model(&models.Check{}).Find(&ids).Error; err != nil {
				return nil, fmt.Errorf("error getting checks with selectors[%v]: %w", resourceSelector, err)
			}

			if len(ids) == 0 {
				cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
			} else {
				cacheToUse.SetDefault(hash, ids)
			}

			allChecks = append(allChecks, ids...)
		}
	}

	return allChecks, nil
}

func GetChecksByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Check, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var checks []models.Check
	err := ctx.DB().Where(LocalFilter).Where("id IN ?", ids).Find(&checks).Error
	return checks, err
}
