package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

var (
	allowedColumnFieldsInComponents = []string{
		"owner",
		"topology_type",
		"topology_id",
		"parent_id",
		"type", // Deprecated. Use resource_selector.types instead
	}
)

func GetComponentsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Component, error) {
	var components []models.Component
	for i := range ids {
		c, err := ComponentFromCache(ctx, ids[i].String())
		if err != nil {
			return nil, err
		}

		components = append(components, c)
	}

	return components, nil
}

func FindComponents(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]models.Component, error) {
	items, err := FindComponentIDs(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetComponentsByIDs(ctx, items)
}

func FindComponentIDs(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allComponents []uuid.UUID
	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector(ctx, resourceSelector, "components", "labels", allowedColumnFieldsInComponents)
		if err != nil {
			return nil, err
		}

		allComponents = append(allComponents, items...)
	}

	return allComponents, nil
}
