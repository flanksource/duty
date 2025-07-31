package query

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	gocache "github.com/patrickmn/go-cache"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
)

func FlushConfigCache(ctx context.Context) error {
	return configItemRelatedTypeCache.Clear(ctx)
}

// <id>/<related_type> -> []related_ids
var configItemRelatedTypeCache = cache.New[[]string](gocache_store.NewGoCache(gocache.New(10*time.Minute, 10*time.Minute)))

func configItemRelatedTypeCacheKey(id, typ string) string {
	return "configRelatedType:" + id + typ
}

// <id> -> models.ConfigItem
var configItemCache = cache.New[models.ConfigItem](gocache_store.NewGoCache(gocache.New(10*time.Minute, 10*time.Minute)))

func configItemCacheKey(id string) string {
	return "configID:" + id
}

// <id> -> models.ConfigItemSummary
var configItemSummaryCache = cache.New[models.ConfigItemSummary](gocache_store.NewGoCache(gocache.New(10*time.Minute, 10*time.Minute)))

func configItemSummaryCacheKey(id string) string {
	return "configIDSummary:" + id
}

// <config_id> -> []related_ids
var configRelationCache = cache.New[[]string](gocache_store.NewGoCache(gocache.New(10*time.Minute, 10*time.Minute)))

func configRelationCacheKey(id string) string {
	return "configRelatedIDs:" + id
}

func SyncConfigCache(ctx context.Context) error {
	var configItems []models.ConfigItem
	if err := ctx.DB().Table("config_items").Where(LocalFilter).Find(&configItems).Error; err != nil {
		return fmt.Errorf("error querying config items for cache: %w", err)
	}

	// We create a type group to always override type -> configIDs
	configIDTypeMap := make(map[string]string)
	for _, ci := range configItems {
		if err := configItemCache.Set(ctx, configItemCacheKey(ci.ID.String()), ci); err != nil {
			return fmt.Errorf("error caching config(%s): %w", ci.ID, err)
		}

		if ci.Type != nil {
			configIDTypeMap[ci.ID.String()] = *ci.Type
		}
	}

	var configRelations []models.ConfigRelationship
	if err := ctx.DB().Table("config_relationships").Where("deleted_at IS NULL").Find(&configRelations).Error; err != nil {
		return fmt.Errorf("error querying config relationships for cache: %w", err)
	}

	relGroup := make(map[string][]string)
	for _, ci := range configRelations {
		relGroup[ci.ConfigID] = append(relGroup[ci.ConfigID], ci.RelatedID)
	}

	configIDRelatedTypeToRelatedIDs := make(map[string][]string)
	for cID, relIDs := range relGroup {
		if err := configRelationCache.Set(ctx, configRelationCacheKey(cID), relIDs); err != nil {
			return fmt.Errorf("error setting config relationships in cache: %w", err)
		}
		for _, relID := range relIDs {
			configIDRelatedTypeToRelatedIDs[configItemRelatedTypeCacheKey(relID, configIDTypeMap[cID])] = append(configIDRelatedTypeToRelatedIDs[configItemRelatedTypeCacheKey(relID, configIDTypeMap[cID])], cID)
		}
	}

	for cacheKey, relatedIDs := range configIDRelatedTypeToRelatedIDs {
		if err := configItemRelatedTypeCache.Set(ctx, cacheKey, relatedIDs); err != nil {
			return fmt.Errorf("error setting config item in cache: %w", err)
		}
	}
	return nil
}

func ConfigIDsByTypeFromCache(ctx context.Context, id, typ string) ([]string, error) {
	return configItemRelatedTypeCache.Get(ctx, configItemRelatedTypeCacheKey(id, typ))
}

func ConfigItemFromCache(ctx context.Context, id string) (models.ConfigItem, error) {
	c, err := configItemCache.Get(ctx, configItemCacheKey(id))
	if err != nil {
		var cacheErr *store.NotFound
		if !errors.As(err, &cacheErr) {
			return c, err
		}

		var ci models.ConfigItem
		if err := ctx.DB().Where("id = ?", id).Where("deleted_at IS NULL").First(&ci).Error; err != nil {
			return ci, err
		}

		return ci, configItemCache.Set(ctx, configItemCacheKey(id), ci)
	}

	return c, nil
}

func ConfigItemSummaryFromCache(ctx context.Context, id string) (models.ConfigItemSummary, error) {
	c, err := configItemSummaryCache.Get(ctx, configItemSummaryCacheKey(id))
	if err != nil {
		var cacheErr *store.NotFound
		if !errors.As(err, &cacheErr) {
			return c, err
		}

		var ci models.ConfigItemSummary
		if err := ctx.DB().Where("id = ?", id).Where("deleted_at IS NULL").First(&ci).Error; err != nil {
			return ci, err
		}

		return ci, configItemSummaryCache.Set(ctx, configItemSummaryCacheKey(id), ci)
	}

	return c, nil
}

func ConfigRelationsFromCache(ctx context.Context, id string) ([]string, error) {
	return configRelationCache.Get(ctx, configRelationCacheKey(id))
}
