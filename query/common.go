package query

import (
	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
)

func lookupIDs(ctx context.Context, table, namespace, name, componentType string) ([]uuid.UUID, error) {
	if name == "" && namespace == "" && componentType == "" {
		return nil, nil
	}

	var ids []uuid.UUID
	query := ctx.DB().Table(table).Select("id")
	if name != "" {
		query = query.Where("name = ?", name)
	}
	if namespace != "" {
		query = query.Where("namespace = ?", namespace)
	}
	if componentType != "" {
		query = query.Where("type = ?", componentType)
	}
	if err := query.Find(&ids).Error; err != nil {
		return nil, err
	}

	return ids, nil
}
