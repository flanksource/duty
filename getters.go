package duty

import (
	"errors"
	"fmt"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var (
	// getterCache caches the results for all the getters in this file.
	getterCache = cache.New(time.Second*90, time.Minute*5)

	immutableCache = cache.New(cache.NoExpiration, time.Hour*12)
)

var (
	allowedColumnFieldsInComponents = []string{"owner", "topology_type"}
	allowedColumnFieldsInConfigs    = []string{"config_class"}
)

// CleanCache flushes the getter caches.
// Mainly used by unit tests.
func CleanCache() {
	getterCache.Flush()
	immutableCache.Flush()
}

func cacheKey[T any](field, key string) string {
	var v T
	return fmt.Sprintf("%T:%s=%s", v, field, key)
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

type FindOption func(db *gorm.DB)

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

func PickColumns(columns ...string) FindOption {
	return func(db *gorm.DB) {
		if len(columns) == 0 {
			return
		}
		db.Select(columns)
	}
}

func WhereClause(query any, args ...any) FindOption {
	return func(db *gorm.DB) {
		db.Where(query, args...)
	}
}

func apply(db *gorm.DB, opts ...FindOption) *gorm.DB {
	for _, opt := range opts {
		opt(db)
	}
	return db
}

func FindChecks(ctx context.Context, resourceSelectors types.ResourceSelectors) ([]models.Check, error) {
	ids, err := FindCheckIDs(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetChecksByIDs(ctx, ids, PickColumns("id"))
}

func FindCheckIDs(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	for _, rs := range resourceSelectors {
		if rs.FieldSelector != "" {
			return nil, fmt.Errorf("field selector is not supported for checks (%s)", rs.FieldSelector)
		}
	}

	var allChecks []uuid.UUID
	for _, resourceSelector := range resourceSelectors {
		hash := "FindChecks-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			allChecks = append(allChecks, val.([]uuid.UUID)...)
			continue
		}

		if query := resourceSelectorQuery(ctx, resourceSelector, "labels", nil); query != nil {
			var ids []uuid.UUID
			if err := query.Model(&models.Check{}).Find(&ids).Error; err != nil {
				return nil, fmt.Errorf("error getting checks with selectors[%v]: %w", resourceSelector, err)
			}

			if len(ids) == 0 {
				cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
			} else {
				cacheToUse.SetDefault(hash, ids)
			}

			allChecks = append(allChecks, ids...)
		}
	}

	return allChecks, nil
}

func FindComponents(ctx context.Context, resourceSelectors types.ResourceSelectors, opts ...FindOption) ([]models.Component, error) {
	items, err := FindComponentIDs(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetComponentsByIDs(ctx, items, opts...)
}

func FindComponentIDs(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allComponents []uuid.UUID
	for _, resourceSelector := range resourceSelectors {
		hash := "FindComponents-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			allComponents = append(allComponents, val.([]uuid.UUID)...)
			continue
		}

		if query := resourceSelectorQuery(ctx, resourceSelector, "labels", allowedColumnFieldsInComponents); query != nil {
			var ids []uuid.UUID
			if err := query.Model(&models.Component{}).Find(&ids).Error; err != nil {
				return nil, fmt.Errorf("error getting components with selectors[%v]: %w", resourceSelector, err)
			}

			if len(ids) == 0 {
				cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
			} else {
				cacheToUse.SetDefault(hash, ids)
			}

			allComponents = append(allComponents, ids...)
		}
	}

	return allComponents, nil
}

func FindConfigs(ctx context.Context, resourceSelectors types.ResourceSelectors, opts ...FindOption) ([]models.ConfigItem, error) {
	items, err := FindConfigIDs(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigsByIDs(ctx, items, opts...)
}

func FindConfigIDs(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allConfigs []uuid.UUID

	for _, resourceSelector := range resourceSelectors {
		hash := "FindConfigs-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			allConfigs = append(allConfigs, val.([]uuid.UUID)...)
			continue
		}

		if query := resourceSelectorQuery(ctx, resourceSelector, "tags", allowedColumnFieldsInConfigs); query != nil {
			var ids []uuid.UUID
			if err := query.Model(&models.ConfigItem{}).Find(&ids).Error; err != nil {
				return nil, fmt.Errorf("error getting configs with selectors[%v]: %w", resourceSelector, err)
			}

			if len(ids) == 0 {
				cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
			} else {
				cacheToUse.SetDefault(hash, ids)
			}

			allConfigs = append(allConfigs, ids...)
		}
	}

	return allConfigs, nil
}

// resourceSelectorQuery returns an ANDed query from all the fields
func resourceSelectorQuery(ctx context.Context, resourceSelector types.ResourceSelector, labelsColumn string, allowedColumnsAsFields []string) *gorm.DB {
	if resourceSelector.IsEmpty() {
		return nil
	}

	query := ctx.DB().Debug().Select("id").Where("deleted_at IS NULL")

	if resourceSelector.ID != "" {
		query = query.Where("id = ?", resourceSelector.ID)
	}
	if resourceSelector.Name != "" {
		query = query.Where("name = ?", resourceSelector.Name)
	}
	if resourceSelector.Namespace != "" {
		query = query.Where("namespace = ?", resourceSelector.Namespace)
	}
	if len(resourceSelector.Types) != 0 {
		query = query.Where("type IN ?", resourceSelector.Types)
	}
	if len(resourceSelector.Statuses) != 0 {
		query = query.Where("status IN ?", resourceSelector.Statuses)
	}

	if resourceSelector.Agent != "" {
		if resourceSelector.Agent == "self" {
			query = query.Where("agent_id = ?", uuid.Nil)
		} else if uid, err := uuid.Parse(resourceSelector.Agent); err == nil {
			query = query.Where("agent_id = ?", uid)
		} else { // assume it's an agent name
			agent, err := FindCachedAgent(ctx, resourceSelector.Agent)
			if err != nil {
				return nil
			}
			query = query.Where("agent_id = ?", agent.ID)
		}
	}

	if len(resourceSelector.LabelSelector) > 0 {
		labels := collections.SelectorToMap(resourceSelector.LabelSelector)
		var onlyKeys []string
		for k, v := range labels {
			if v == "" {
				onlyKeys = append(onlyKeys, k)
				delete(labels, k)
			}
		}

		query = query.Where(fmt.Sprintf("%s @> ?", labelsColumn), types.JSONStringMap(labels))
		for _, k := range onlyKeys {
			query = query.Where(fmt.Sprintf("%s ?? ?", labelsColumn), k)
		}
	}

	if len(resourceSelector.FieldSelector) > 0 {
		fields := collections.SelectorToMap(resourceSelector.FieldSelector)
		columnWhereClauses := map[string]string{}
		var props models.Properties
		for k, v := range fields {
			if collections.Contains(allowedColumnsAsFields, k) {
				columnWhereClauses[k] = v
			} else {
				props = append(props, &models.Property{Name: k, Text: v})
			}
		}

		if len(columnWhereClauses) > 0 {
			query = query.Where(columnWhereClauses)
		}

		if len(props) > 0 {
			query = query.Where("properties @> ?", props)
		}
	}

	return query
}

func GetChecksByIDs(ctx context.Context, ids []uuid.UUID, opts ...FindOption) ([]models.Check, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var checks []models.Check
	err := apply(ctx.DB().Where(LocalFilter).Where("id IN ?", ids), opts...).Find(&checks).Error
	return checks, err
}

func GetComponentsByIDs(ctx context.Context, ids []uuid.UUID, opts ...FindOption) ([]models.Component, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var components []models.Component
	err := apply(ctx.DB().Where(LocalFilter).Where("id IN ?", ids), opts...).Find(&components).Error
	return components, err
}

func GetConfigsByIDs(ctx context.Context, ids []uuid.UUID, opts ...FindOption) ([]models.ConfigItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var configs []models.ConfigItem
	err := apply(ctx.DB().Where(LocalFilter).Where("id IN ?", ids), opts...).Find(&configs).Error
	return configs, err
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

func findCachedEntity[T any](ctx context.Context, id string) (*T, error) {
	return findEntityByField[T](ctx, "id", id)
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
