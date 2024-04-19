package query

import (
	"database/sql"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

func GetConfigsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ConfigItem, error) {
	var configs []models.ConfigItem
	for i := range ids {
		config, err := ConfigItemFromCache(ctx, ids[i].String())
		if err != nil {
			return nil, err
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func FindConfig(ctx context.Context, query types.ConfigQuery) (*models.ConfigItem, error) {
	res, err := FindConfigsByResourceSelector(ctx, query.ToResourceSelector())
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return &res[0], nil
}

func FindConfigs(ctx context.Context, config types.ConfigQuery) ([]models.ConfigItem, error) {
	return FindConfigsByResourceSelector(ctx, config.ToResourceSelector())
}

func FindConfigIDs(ctx context.Context, config types.ConfigQuery) ([]uuid.UUID, error) {
	return FindConfigIDsByResourceSelector(ctx, config.ToResourceSelector())
}

func FindConfigsByResourceSelector(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]models.ConfigItem, error) {
	items, err := FindConfigIDsByResourceSelector(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigsByIDs(ctx, items)
}

func FindConfigIDsByResourceSelector(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allConfigs []uuid.UUID

	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector(ctx, resourceSelector, "config_items", models.AllowedColumnFieldsInConfigs)
		if err != nil {
			return nil, err
		}

		allConfigs = append(allConfigs, items...)
	}

	return allConfigs, nil
}

func FindConfigForComponent(ctx context.Context, componentID, configType string) ([]models.ConfigItem, error) {
	db := ctx.DB()
	relationshipQuery := db.Table("config_component_relationships").
		Select("config_id").
		Where("component_id = ? AND deleted_at IS NULL", componentID)
	query := db.Table("config_items").Where("id IN (?)", relationshipQuery)
	if configType != "" {
		query = query.Where("type = @config_type OR config_class = @config_type", sql.Named("config_type", configType))
	}
	var dbConfigObjects []models.ConfigItem
	err := query.Find(&dbConfigObjects).Error
	return dbConfigObjects, err
}
