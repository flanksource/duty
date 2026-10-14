package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// connectionDetailsView is where connections are read from: the PostgREST roles that row-level security runs as
// can't read the connections table, only its masked view (#2114).
const connectionDetailsView = "connection_details"

func FindConnectionIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, connectionDetailsView, limit, resourceSelectors...)
}

// FindConnectionsByResourceSelector returns the connections the selectors match, with their credentials masked.
func FindConnectionsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.Connection, error) {
	ids, err := FindConnectionIDsByResourceSelector(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	} else if len(ids) == 0 {
		return nil, nil
	}

	var connections []models.Connection
	if err := ctx.DB().Table(connectionDetailsView).Where("id IN ?", ids).Find(&connections).Error; err != nil {
		return nil, err
	}

	return connections, nil
}
