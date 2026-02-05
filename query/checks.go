package query

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func FindChecks(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.Check, error) {
	ids, err := FindCheckIDs(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetChecksByIDs(ctx, ids)
}

func FindCheckIDs(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	for _, rs := range resourceSelectors {
		if rs.FieldSelector != "" {
			return nil, fmt.Errorf("field selector is not supported for checks (%s)", rs.FieldSelector)
		}
	}

	return queryTableWithResourceSelectors(ctx, "checks", limit, resourceSelectors...)
}

func GetChecksByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Check, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var checks []models.Check
	err := ctx.DB().Where("deleted_at IS NULL").Where("id IN ?", ids).Find(&checks).Error
	return checks, err
}
