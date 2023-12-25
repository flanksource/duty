package query

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	gocache "github.com/patrickmn/go-cache"
)

func configQuery(db *gorm.DB, config types.ConfigQuery) *gorm.DB {
	query := db.Table("config_items").Where("agent_id = '00000000-0000-0000-0000-000000000000'")
	if config.Class != "" {
		query = query.Where("config_class = ?", config.Class)
	}
	if config.Name != "" {
		query = query.Where("name = ?", config.Name)
	}
	if config.Namespace != "" {
		query = query.Where("namespace = ?", config.Namespace)
	}

	if config.Tags != nil && len(config.Tags) > 0 {
		query = query.Where("tags @> ?", config.Tags)
	}

	// Type is derived from v1.Config.Type which is a user input field
	// It can refer to both type or config_class for now
	if config.Type != "" {
		query = query.Where("type = @config_type OR config_class = @config_type", sql.Named("config_type", config.Type))
	}
	if len(config.ExternalID) > 0 {
		query = query.Where("external_id @> ?", config.ExternalID)
	}

	if len(config.ID) > 0 {
		query = query.Where("id @> ?", config.ID)

	}
	return query
}

var configCache = gocache.New(30*time.Minute, 1*time.Hour)

func FindConfig(ctx context.Context, config types.ConfigQuery) (*models.ConfigItem, error) {
	if ctx.DB() == nil {
		logger.Debugf("Config lookup on %v will be ignored, db not initialized", config)
		return nil, gorm.ErrRecordNotFound
	}

	cacheKey := config.Hash()
	if cacheKey == "" {
		return nil, fmt.Errorf("error generating cacheKey for %s", config)
	}

	if val, exists := configCache.Get(cacheKey); exists {
		// If config item is not found, it is stored as nil
		if val == nil {
			return nil, nil
		}
		return val.(*models.ConfigItem), nil
	}

	var item models.ConfigItem
	query := configQuery(ctx.DB(), config)
	tx := query.Limit(1).Find(&item)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		// If config item is not found, stored as nil for a short duration
		configCache.Set(cacheKey, nil, 10*time.Minute)
		return nil, nil
	}

	configCache.Set(cacheKey, &item, gocache.DefaultExpiration)
	return &item, nil
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
