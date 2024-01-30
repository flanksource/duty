package query

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	gocache "github.com/patrickmn/go-cache"

	"github.com/eko/gocache/lib/v4/cache"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
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

func FindConfigs(db *gorm.DB, config types.ConfigQuery) ([]models.ConfigItem, error) {
	configHash := config.Hash()
	if configHash == "" {
		return nil, fmt.Errorf("error generating cacheKey for %s", config)
	}
	cacheKey := "FindConfigs" + configHash

	if val, exists := configCache.Get(cacheKey); exists {
		// If config items are not found, it is stored as nil
		if val == nil {
			return nil, nil
		}
		return val.([]models.ConfigItem), nil
	}

	var items []models.ConfigItem
	tx := configQuery(db, config).Find(&items)
	if tx.Error != nil {
		return nil, fmt.Errorf("error querying config items with query(%v) err: %w", config, tx.Error)
	}
	if tx.RowsAffected == 0 {
		// If config item is not found, stored as nil for a short duration
		configCache.Set(cacheKey, nil, 10*time.Minute)
		return nil, nil
	}

	configCache.Set(cacheKey, items, gocache.DefaultExpiration)
	return items, nil
}

func FindConfigIDs(db *gorm.DB, config types.ConfigQuery) ([]uuid.UUID, error) {
	configHash := config.Hash()
	if configHash == "" {
		return nil, fmt.Errorf("error generating cacheKey for %s", config)
	}
	cacheKey := "FindConfigIDs" + configHash

	if val, exists := configCache.Get(cacheKey); exists {
		// If config items are not found, it is stored as nil
		if val == nil {
			return nil, nil
		}
		return val.([]uuid.UUID), nil
	}

	var items []uuid.UUID
	tx := configQuery(db, config).Select("id").Find(&items)
	if tx.Error != nil {
		return nil, fmt.Errorf("error querying config items with query(%v) err: %w", config, tx.Error)
	}
	if tx.RowsAffected == 0 {
		// If config item is not found, stored as nil for a short duration
		configCache.Set(cacheKey, nil, 10*time.Minute)
		return nil, nil
	}

	configCache.Set(cacheKey, items, gocache.DefaultExpiration)
	return items, nil
}

func FindConfig(db *gorm.DB, config types.ConfigQuery) (*models.ConfigItem, error) {
	if db == nil {
		return nil, fmt.Errorf("db not initialized")
	}

	configHash := config.Hash()
	if configHash == "" {
		return nil, fmt.Errorf("error generating cacheKey for %s", config)
	}
	cacheKey := "FindConfig" + configHash

	if val, exists := configCache.Get(cacheKey); exists {
		// If config item is not found, it is stored as nil
		if val == nil {
			return nil, nil
		}
		return val.(*models.ConfigItem), nil
	}

	var item models.ConfigItem
	query := configQuery(db, config)
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

// <type> -> []ids
var configItemTypeCache = cache.New[[]string](gocache_store.NewGoCache(gocache.New(6*time.Hour, 6*time.Hour)))

func configItemTypeCacheKey(typ string) string {
	return "configType:" + typ
}

// <id> -> models.ConfigItem
var configItemCache = cache.New[models.ConfigItem](gocache_store.NewGoCache(gocache.New(6*time.Hour, 6*time.Hour)))

func configItemCacheKey(id string) string {
	return "configID:" + id
}

// <config_id> -> []related_ids
var configRelationCache = cache.New[[]string](gocache_store.NewGoCache(gocache.New(6*time.Hour, 6*time.Hour)))

func configRelationCacheKey(id string) string {
	return "configRelatedIDs:" + id
}

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

func SyncConfigCache(ctx context.Context) error {
	var configItems []models.ConfigItem
	if err := ctx.DB().Table("config_items").Where(LocalFilter).FindInBatches(&configItems, 1000, func(*gorm.DB, int) error { return nil }).Error; err != nil {
		return fmt.Errorf("error querying config items for cache: %w", err)
	}

	// We create a type group to always override type -> configIDs
	typeGroup := make(map[string][]string)
	for _, ci := range configItems {
		if err := configItemCache.Set(ctx, configItemCacheKey(ci.ID.String()), ci); err != nil {
			return fmt.Errorf("error setting config item in cache: %w", err)
		}

		if ci.Type != nil {
			typeGroup[*ci.Type] = append(typeGroup[*ci.Type], ci.ID.String())
		}
	}

	for typ, configIDs := range typeGroup {
		if err := configItemTypeCache.Set(ctx, configItemTypeCacheKey(typ), configIDs); err != nil {
			return fmt.Errorf("error setting config item in cache: %w", err)
		}
	}

	var configRelations []models.ConfigRelationship
	if err := ctx.DB().Table("config_relationships").Where("deleted_at IS NULL").FindInBatches(&configRelations, 5000, func(*gorm.DB, int) error { return nil }).Error; err != nil {
		return fmt.Errorf("error querying config relationships for cache: %w", err)
	}

	relGroup := make(map[string][]string)
	for _, ci := range configRelations {
		relGroup[ci.ConfigID] = append(relGroup[ci.ConfigID], ci.RelatedID)
	}

	// TODO: Acquire Lock ? Old relationships can persist
	// configRelationCache.Clear(ctx)

	for ciD, relIDs := range relGroup {
		if err := configRelationCache.Set(ctx, configRelationCacheKey(ciD), relIDs); err != nil {
			return fmt.Errorf("error setting config relationships in cache: %w", err)
		}
	}

	return nil
}

func ConfigIDsByTypeFromCache(ctx context.Context, typ string) ([]string, error) {
	return configItemTypeCache.Get(ctx, configItemTypeCacheKey(typ))
}

func ConfigItemFromCache(ctx context.Context, id string) (models.ConfigItem, error) {
	return configItemCache.Get(ctx, configItemCacheKey(id))
}

func ConfigRelationsFromCache(ctx context.Context, id string) ([]string, error) {
	return configRelationCache.Get(ctx, configRelationCacheKey(id))
}
