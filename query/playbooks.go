package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func FindPlaybookIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, "playbooks", limit, resourceSelectors...)
}

func FindPlaybooksByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.Playbook, error) {
	ids, err := queryTableWithResourceSelectors(ctx, "playbooks", limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	var playbooks []models.Playbook
	if err := ctx.DB().Where("id IN ?", ids).Find(&playbooks).Error; err != nil {
		return nil, err
	}

	return playbooks, nil
}
