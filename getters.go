package duty

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// getterCache caches the results for all the getters in this file.
var getterCache = cache.New(time.Second*90, time.Minute*5)

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

func FindCachedPerson(ctx dbContext, agentID string) (*models.Person, error) {
	person, err := findCachedEntity[models.Person](ctx, agentID)
	if err != nil {
		return nil, err
	}

	return person, nil
}

func FindCacheComponent(ctx dbContext, componentID string) (*models.Component, error) {
	component, err := findCachedEntity[models.Component](ctx, componentID)
	if err != nil {
		return nil, err
	}

	return component, nil
}

func FindCachedConfig(ctx dbContext, configID string) (*models.ConfigItem, error) {
	config, err := findCachedEntity[models.ConfigItem](ctx, configID)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func findCachedEntity[T any](ctx dbContext, id string) (*T, error) {
	if value, ok := getterCache.Get(id); ok {
		if cache, ok := value.(*T); ok {
			return cache, nil
		} else {
			return nil, fmt.Errorf("unexpected cached value type: %T", value)
		}
	}

	var resource T
	if err := ctx.DB().Where("id = ?", id).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	getterCache.SetDefault(id, &resource)
	return &resource, nil
}
