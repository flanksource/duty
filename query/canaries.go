package query

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func FindCanaries(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.Canary, error) {
	ids, err := FindCanaryIDs(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetCanariesByIDs(ctx, ids)
}

func FindCanaryIDs(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	for _, rs := range resourceSelectors {
		if rs.FieldSelector != "" {
			return nil, fmt.Errorf("field selector is not supported for canaries (%s)", rs.FieldSelector)
		}
	}

	return queryTableWithResourceSelectors(ctx, "canaries", limit, resourceSelectors...)
}

func GetCanariesByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Canary, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var canaries []models.Canary
	err := ctx.DB().Where("deleted_at IS NULL").Where("id IN ?", ids).Find(&canaries).Error
	return canaries, err
}