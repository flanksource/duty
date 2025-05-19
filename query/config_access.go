package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

func FindConfigAccessByConfigIDs(ctx context.Context, configIDs []uuid.UUID) ([]models.ConfigAccessSummary, error) {
	var configAccess []models.ConfigAccessSummary
	if err := ctx.DB().
		Where("config_id IN (?)", configIDs).
		Find(&configAccess).Error; err != nil {
		return nil, err
	}

	return configAccess, nil
}
