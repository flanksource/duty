package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func FindConnectionIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, "connections", limit, resourceSelectors...)
}

func FindConnectionsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.Connection, error) {
	ids, err := queryTableWithResourceSelectors(ctx, "connections", limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	var connections []models.Connection
	if err := ctx.DB().Where("id IN ?", ids).Find(&connections).Error; err != nil {
		return nil, err
	}

	return connections, nil
}
