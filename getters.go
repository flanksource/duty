package duty

import (
	"errors"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/models"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// getterCache caches the results for all the getters in this file.
var getterCache = cache.New(time.Second*30, time.Minute*5)

type cachedItem struct {
	resource any
	notFound bool
}

func FindCachedAgent(ctx dbContext, agentID string) (*models.Agent, error) {
	agent, err := findCachedEntity[models.Agent](ctx, agentID)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func FindCachedCheck(ctx dbContext, agentID string) (*models.Check, error) {
	check, err := findCachedEntity[models.Check](ctx, agentID)
	if err != nil {
		return nil, err
	}

	return check, nil
}

func FindCachedCanary(ctx dbContext, agentID string) (*models.Canary, error) {
	canary, err := findCachedEntity[models.Canary](ctx, agentID)
	if err != nil {
		return nil, err
	}

	return canary, nil
}

func findCachedEntity[T any](ctx dbContext, id string) (*T, error) {
	if value, ok := getterCache.Get(id); ok {
		if cache, ok := value.(cachedItem); ok {
			if cache.notFound {
				return nil, nil
			}

			return cache.resource.(*T), nil
		} else {
			logger.Warnf("Unexpected cached value type: %T", value)
		}
	}

	var resource T
	if err := ctx.DB().Where("id = ?", id).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			getterCache.SetDefault(id, cachedItem{notFound: true})
			return nil, nil
		}

		return nil, err
	}

	getterCache.SetDefault(id, cachedItem{resource: &resource})
	return &resource, nil
}
