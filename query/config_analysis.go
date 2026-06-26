package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func FindConfigAnalysisByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.ConfigAnalysis, error) {
	ids, err := FindConfigAnalysisIDsByResourceSelector(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigAnalysisByIDs(ctx, ids)
}

func FindConfigAnalysisIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, configAnalysisItemsView, limit, resourceSelectors...)
}

func GetConfigAnalysisByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ConfigAnalysis, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var analyses []models.ConfigAnalysis
	if err := ctx.DB().Where("id IN ?", ids).Find(&analyses).Error; err != nil {
		return nil, err
	}

	return analyses, nil
}
