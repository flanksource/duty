package query

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	gocache "github.com/patrickmn/go-cache"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
)

func FlushComponentCache(ctx context.Context) error {
	return componentCache.Clear(ctx)
}

const cacheJobBatchSize = 1000

// <id> -> models.Component
var componentCache = cache.New[models.Component](gocache_store.NewGoCache(gocache.New(10*time.Minute, 10*time.Minute)))

func componentCacheKey(id string) string {
	return "componentID:" + id
}

func SyncComponentCache(ctx context.Context) error {
	var next string
	for {
		var components []models.Component
		if err := ctx.DB().Where("deleted_at IS NULL").Where("id > ?", next).Limit(cacheJobBatchSize).Find(&components).Error; err != nil {
			return fmt.Errorf("error querying components for cache: %w", err)
		}

		if len(components) == 0 {
			break
		}

		for _, ci := range components {
			if err := componentCache.Set(ctx, componentCacheKey(ci.ID.String()), ci); err != nil {
				return fmt.Errorf("error caching component(%s): %w", ci.ID, err)
			}
		}

		next = components[len(components)-1].ID.String()
	}

	return nil
}

func ComponentFromCache(ctx context.Context, id string) (models.Component, error) {
	c, err := componentCache.Get(ctx, componentCacheKey(id))
	if err != nil {
		var cacheErr *store.NotFound
		if !errors.As(err, &cacheErr) {
			return c, err
		}

		var component models.Component
		if err := ctx.DB().Where("id = ?", id).Where("deleted_at IS NULL").First(&component).Error; err != nil {
			return component, err
		}

		return component, componentCache.Set(ctx, componentCacheKey(id), component)
	}

	return c, nil
}

var SyncComponentCacheJob = &job.Job{
	Name:       "SyncComponentCache",
	Schedule:   "@every 5m",
	JobHistory: true,
	Retention:  job.RetentionHour,
	Fn: func(ctx job.JobRuntime) error {
		return SyncComponentCache(ctx.Context)
	},
}
