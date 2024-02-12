package query

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
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

// <name/type/parentID/topologyID> -> models.Component
var componentNameTypeParentTopologyCache = cache.New[models.Component](gocache_store.NewGoCache(gocache.New(30*time.Minute, 30*time.Minute)))

func componentCacheKey(id string) string {
	return "componentID:" + id
}

func componentNameTypeParentTopologyCacheKey(name, typ string, parentID *uuid.UUID, topologyID string) string {
	pID := uuid.Nil.String()
	if parentID != nil {
		pID = parentID.String()
	}
	return name + typ + pID + topologyID
}

func SyncComponentCache(ctx context.Context) error {
	next := uuid.Nil.String()
	for {
		var components []models.Component
		if err := ctx.DB().Where("agent_id = ?", uuid.Nil).Where("id > ?", next).Limit(cacheJobBatchSize).Find(&components).Error; err != nil {
			return fmt.Errorf("error querying components for cache: %w", err)
		}

		if len(components) == 0 {
			break
		}

		for _, comp := range components {
			if err := componentCache.Set(ctx, componentCacheKey(comp.ID.String()), comp); err != nil {
				return fmt.Errorf("error caching component(%s): %w", comp.ID, err)
			}

			cacheKey := componentNameTypeParentTopologyCacheKey(comp.Name, comp.Type, comp.ParentId, comp.TopologyID.String())
			if err := componentNameTypeParentTopologyCache.Set(ctx, cacheKey, comp); err != nil {
				return fmt.Errorf("error caching component(%s): %w", comp.ID, err)
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
			return c, fmt.Errorf("error fetching component[%s] from cache: %w", id, err)
		}

		var component models.Component
		if err := ctx.DB().Where("id = ?", id).Where("deleted_at IS NULL").First(&component).Error; err != nil {
			return component, err
		}

		return component, componentCache.Set(ctx, componentCacheKey(id), component)
	}

	return c, nil
}

func ComponentFromByNameTypeParentTopology(ctx context.Context, name, typ string, parentID *uuid.UUID, topologyID string) (models.Component, error) {
	cacheKey := componentNameTypeParentTopologyCacheKey(name, typ, parentID, topologyID)
	return componentNameTypeParentTopologyCache.Get(ctx, cacheKey)
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
