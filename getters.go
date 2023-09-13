package duty

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// getterCache caches the results for all the getters in this file.
var getterCache = cache.New(time.Second*90, time.Minute*5)

func FindCachedAgent(ctx DBContext, id string) (*models.Agent, error) {
	if id == uuid.Nil.String() {
		return nil, nil
	}

	agent, err := findCachedEntity[models.Agent](ctx, id)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func FindCachedCheck(ctx DBContext, id string) (*models.Check, error) {
	check, err := findCachedEntity[models.Check](ctx, id)
	if err != nil {
		return nil, err
	}

	return check, nil
}

func FindCachedCanary(ctx DBContext, id string) (*models.Canary, error) {
	canary, err := findCachedEntity[models.Canary](ctx, id)
	if err != nil {
		return nil, err
	}

	return canary, nil
}

func FindCachedPerson(ctx DBContext, id string) (*models.Person, error) {
	person, err := findCachedEntity[models.Person](ctx, id)
	if err != nil {
		return nil, err
	}

	return person, nil
}

func FindPersonByEmail(ctx DBContext, email string) (*models.Person, error) {
	person, err := findEntityByField[models.Person](ctx, "email", email, false)
	if err != nil {
		return nil, err
	}

	return person, nil
}

func FindTeamByName(ctx DBContext, name string) (*models.Team, error) {
	team, err := findEntityByField[models.Team](ctx, "name", name, false)
	if err != nil {
		return nil, err
	}

	return team, nil
}

func FindCachedComponent(ctx DBContext, id string) (*models.Component, error) {
	component, err := findCachedEntity[models.Component](ctx, id)
	if err != nil {
		return nil, err
	}

	return component, nil
}

func FindCachedConfig(ctx DBContext, id string) (*models.ConfigItem, error) {
	config, err := findCachedEntity[models.ConfigItem](ctx, id)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func FindCachedIncident(ctx DBContext, id string) (*models.Incident, error) {
	incident, err := findCachedEntity[models.Incident](ctx, id)
	if err != nil {
		return nil, err
	}

	return incident, nil
}

func findCachedEntity[T any](ctx DBContext, id string) (*T, error) {
	return findEntityByField[T](ctx, "id", id, true)
}

func findEntityByField[T any](ctx DBContext, field, key string, withCache bool) (*T, error) {
	if withCache {
		if value, ok := getterCache.Get(key); ok {
			if cache, ok := value.(*T); ok {
				return cache, nil
			} else {
				return nil, fmt.Errorf("unexpected cached value type: %T", value)
			}
		}
	}

	var resource T
	if err := ctx.DB().Where(fmt.Sprintf("%s = ?", field), key).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	getterCache.SetDefault(key, &resource)
	return &resource, nil
}
