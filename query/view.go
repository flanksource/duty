package query

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

func FindViews(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.View, error) {
	return FindViewsByResourceSelector(ctx, limit, resourceSelectors...)
}

func FindViewsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.View, error) {
	items, err := FindViewIDsByResourceSelector(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetViewsByIDs(ctx, items)
}

func FindViewIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, models.View{}.TableName(), limit, resourceSelectors...)
}

func GetViewsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.View, error) {
	var views []models.View
	if len(ids) == 0 {
		return views, nil
	}

	if err := ctx.DB().Where("id IN ?", ids).Find(&views).Error; err != nil {
		return nil, err
	}

	return views, nil
}
