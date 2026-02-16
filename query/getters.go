package query

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var (
	// getterCache caches the results for all the getters in this file.
	getterCache = cache.New(time.Second*90, time.Minute*5)

	immutableCache = cache.New(cache.NoExpiration, time.Hour*12)

	scopeCache = cache.New(time.Hour, time.Hour*2)
)

func FlushGettersCache() {
	getterCache.Flush()
	immutableCache.Flush()
}

// InvalidateCacheByID deletes a single item from the getters cache
func InvalidateCacheByID[T any](id string) {
	key := cacheKey[T]("id", id)
	getterCache.Delete(key)
}

type GetterOption uint8

const (
	GetterOptionNoCache GetterOption = iota + 1
)

type GetterOptions []GetterOption

func (t GetterOptions) IsSet(option GetterOption) bool {
	for _, opt := range t {
		if opt == option {
			return true
		}
	}

	return false
}

func findCachedEntity[T any](ctx context.Context, id string) (*T, error) {
	return findEntityByField[T](ctx, "id", id)
}

func cacheKey[T any](field, key string) string {
	var v T
	return fmt.Sprintf("%T:%s=%s", v, field, key)
}

func findEntityByField[T any](ctx context.Context, field, key string, opts ...GetterOption) (*T, error) {
	if !GetterOptions(opts).IsSet(GetterOptionNoCache) {
		if value, ok := getterCache.Get(cacheKey[T](field, key)); ok {
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

	getterCache.SetDefault(cacheKey[T](field, key), &resource)
	return &resource, nil
}

func GetCachedComponent(ctx context.Context, id string) (*models.Component, error) {
	component, err := findCachedEntity[models.Component](ctx, id)
	if err != nil {
		return nil, err
	}

	return component, nil
}

func GetCachedConfig(ctx context.Context, id string) (*models.ConfigItem, error) {
	config, err := findCachedEntity[models.ConfigItem](ctx, id)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func GetCachedIncident(ctx context.Context, id string) (*models.Incident, error) {
	incident, err := findCachedEntity[models.Incident](ctx, id)
	if err != nil {
		return nil, err
	}

	return incident, nil
}

func FindCachedAgent(ctx context.Context, identifier string) (*models.Agent, error) {
	var field string
	if _, err := uuid.Parse(identifier); err == nil {
		field = "id"
	} else {
		field = "name"
	}

	agent, err := findEntityByField[models.Agent](ctx, field, identifier)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func FindCachedCheck(ctx context.Context, id string) (*models.Check, error) {
	check, err := findCachedEntity[models.Check](ctx, id)
	if err != nil {
		return nil, err
	}

	return check, nil
}

func FindCachedCanary(ctx context.Context, id string) (*models.Canary, error) {
	canary, err := findCachedEntity[models.Canary](ctx, id)
	if err != nil {
		return nil, err
	}

	return canary, nil
}

// FindHumanPeople returns all people that represent real users.
// It excludes agents and access tokens (which have a non-null type) and
// system users (which have no email).
func FindHumanPeople(ctx context.Context) ([]models.Person, error) {
	var people []models.Person
	if err := ctx.DB().
		Where("deleted_at IS NULL").
		Where("type IS NULL").
		Where("email IS NOT NULL").
		Find(&people).Error; err != nil {
		return nil, err
	}

	return people, nil
}

// FindPerson looks up a person by the given identifier which can either be
//   - UUID
//   - email
func FindPerson(ctx context.Context, identifier string, opts ...GetterOption) (*models.Person, error) {
	var field string
	if _, err := uuid.Parse(identifier); err == nil {
		field = "id"
	} else {
		field = "email"
	}

	person, err := findEntityByField[models.Person](ctx, field, identifier, opts...)
	if err != nil {
		return nil, err
	}

	return person, nil
}

// FindTeam looks up a team by the given identifier which can either be
//   - UUID
//   - team name
func FindTeam(ctx context.Context, identifier string, opts ...GetterOption) (*models.Team, error) {
	var field string
	if _, err := uuid.Parse(identifier); err == nil {
		field = "id"
	} else {
		field = "name"
	}

	team, err := findEntityByField[models.Team](ctx, field, identifier, opts...)
	if err != nil {
		return nil, err
	}

	return team, nil
}

func FindPlaybook(ctx context.Context, identifier string, opts ...GetterOption) (*models.Playbook, error) {
	var field string
	if _, err := uuid.Parse(identifier); err == nil {
		field = "id"
	} else {
		field = "name"
	}

	team, err := findEntityByField[models.Playbook](ctx, field, identifier, opts...)
	if err != nil {
		return nil, err
	}

	return team, nil
}
