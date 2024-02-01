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
	"github.com/samber/lo"
	"gorm.io/gorm"
)

var (
	// getterCache caches the results for all the getters in this file.
	getterCache = cache.New(time.Second*90, time.Minute*5)

	immutableCache = cache.New(cache.NoExpiration, time.Hour*12)
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

// FindPerson looks up a person by the given identifier which can either be
//   - UUID
//   - email
func FindPerson(ctx DBContext, identifier string, opts ...GetterOption) (*models.Person, error) {
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
func FindTeam(ctx DBContext, identifier string, opts ...GetterOption) (*models.Team, error) {
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

func FindCheckIDs(ctx context.Context, resourceSelector types.ResourceSelector) ([]models.Check, error) {
	return FindChecks(ctx, []types.ResourceSelector{resourceSelector}, PickColumns("id"))
}

func FindChecks(ctx context.Context, resourceSelectors types.ResourceSelectors, opts ...FindOption) ([]models.Check, error) {
	for _, rs := range resourceSelectors {
		if rs.FieldSelector != "" {
			return nil, fmt.Errorf("field selector is not supported for checks (%s)", rs.FieldSelector)
		}
	}

	var allChecks []models.Check
	for _, resourceSelector := range resourceSelectors {
		hash := "FindChecks-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			checks, err := FindChecksByIDs(ctx, val.([]string), opts...)
			if err != nil {
				return nil, err
			}

			allChecks = append(allChecks, checks...)
			continue
		}

		var uniqueChecks []models.Check
		selectorOpts := opts

		if query := firstResourceSelectorQuery(ctx, "checks", resourceSelector); query != nil {
			var checks []models.Check
			if err := apply(query, opts...).Find(&checks).Error; err != nil {
				return nil, fmt.Errorf("error getting checks with selectors[%v]: %w", resourceSelector, err)
			}

			uniqueChecks = checks
			selectorOpts = append(selectorOpts, WhereClause("id::text in ?", lo.Map(
				checks,
				func(c models.Check, _ int) string { return c.ID.String() }),
			))
		}

		if resourceSelector.LabelSelector != "" {
			checks, err := findResourcesByLabels[models.Check](ctx, resourceSelector.LabelSelector, selectorOpts...)
			if err != nil {
				return nil, fmt.Errorf("error getting checks with label selectors[%s]: %w", resourceSelector.LabelSelector, err)
			}

			uniqueChecks = checks
		}

		ids := lo.Map(uniqueChecks, func(c models.Check, _ int) string { return c.ID.String() })
		if len(ids) == 0 {
			cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
		} else {
			cacheToUse.SetDefault(hash, ids)
		}

		allChecks = append(allChecks, uniqueChecks...)
	}

	return lo.UniqBy(allChecks, models.CheckID), nil
}

func FindComponentIDs(ctx context.Context, resourceSelector types.ResourceSelector) ([]models.Component, error) {
	return FindComponents(ctx, []types.ResourceSelector{resourceSelector}, PickColumns("id"))
}

func FindComponents(ctx context.Context, resourceSelectors types.ResourceSelectors, opts ...FindOption) ([]models.Component, error) {
	var allComponents []models.Component
	for _, resourceSelector := range resourceSelectors {
		hash := "FindComponents-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			components, err := FindComponentsByIDs(ctx, val.([]string), opts...)
			if err != nil {
				return nil, err
			}

			allComponents = append(allComponents, components...)
			continue
		}

		var uniqueComponents []models.Component
		selectorOpts := opts

		if query := firstResourceSelectorQuery(ctx, "components", resourceSelector); query != nil {
			var components []models.Component
			if err := apply(query, opts...).Find(&components).Error; err != nil {
				return nil, fmt.Errorf("error getting components with selectors[%v]: %w", resourceSelector, err)
			}

			uniqueComponents = components
			selectorOpts = append(selectorOpts, WhereClause("id::text in ?", lo.Map(
				components,
				func(c models.Component, _ int) string { return c.ID.String() }),
			))
		}

		if resourceSelector.LabelSelector != "" {
			labelComponents, err := findResourcesByLabels[models.Component](ctx, resourceSelector.LabelSelector, selectorOpts...)
			if err != nil {
				return nil, fmt.Errorf("error getting components with label selectors[%s]: %w", resourceSelector.LabelSelector, err)
			}
			uniqueComponents = labelComponents
			selectorOpts = append(selectorOpts, WhereClause("id::text in ?", lo.Map(
				labelComponents,
				func(c models.Component, _ int) string { return c.ID.String() }),
			))
		}

		if resourceSelector.FieldSelector != "" {
			fieldComponents, err := FindComponentsByField(ctx, resourceSelector.FieldSelector, selectorOpts...)
			if err != nil {
				return nil, fmt.Errorf("error getting components with field selectors[%s]: %w", resourceSelector.FieldSelector, err)
			}
			uniqueComponents = fieldComponents
		}

		ids := lo.Map(uniqueComponents, func(c models.Component, _ int) string { return c.ID.String() })
		if len(ids) == 0 {
			cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
		} else {
			cacheToUse.SetDefault(hash, ids)
		}

		allComponents = append(allComponents, uniqueComponents...)
	}

	return lo.UniqBy(allComponents, models.ComponentID), nil
}

func FindConfigIDs(ctx context.Context, resourceSelector types.ResourceSelector) ([]models.ConfigItem, error) {
	return FindConfigs(ctx, []types.ResourceSelector{resourceSelector}, PickColumns("id"))
}

func FindConfigs(ctx context.Context, resourceSelectors types.ResourceSelectors, opts ...FindOption) ([]models.ConfigItem, error) {
	var allConfigs []models.ConfigItem
	for _, resourceSelector := range resourceSelectors {
		hash := "FindConfigs-CachePrefix" + resourceSelector.Hash()
		cacheToUse := getterCache
		if resourceSelector.Immutable() {
			cacheToUse = immutableCache
		}

		if val, ok := cacheToUse.Get(hash); ok {
			configs, err := FindConfigsByIDs(ctx, val.([]string), opts...)
			if err != nil {
				return nil, err
			}

			allConfigs = append(allConfigs, configs...)
			continue
		}

		var uniqueConfigs []models.ConfigItem
		selectorOpts := opts

		if query := firstResourceSelectorQuery(ctx, "config_items", resourceSelector); query != nil {
			var configs []models.ConfigItem
			if err := apply(query, opts...).Find(&configs).Error; err != nil {
				return nil, fmt.Errorf("error getting configs with selectors[%v]: %w", resourceSelector, err)
			}

			uniqueConfigs = configs
			selectorOpts = append(selectorOpts, WhereClause("id::text in ?",
				lo.Map(configs, func(c models.ConfigItem, _ int) string { return c.ID.String() }),
			))
		}

		if resourceSelector.LabelSelector != "" {
			configs, err := findResourcesByLabels[models.ConfigItem](ctx, resourceSelector.LabelSelector, selectorOpts...)
			if err != nil {
				return nil, fmt.Errorf("error getting configs with label selectors[%s]: %w", resourceSelector.LabelSelector, err)
			}
			uniqueConfigs = configs
			selectorOpts = append(selectorOpts, WhereClause("id::text in ?", lo.Map(
				configs,
				func(c models.ConfigItem, _ int) string { return c.ID.String() }),
			))
		}

		if resourceSelector.FieldSelector != "" {
			configs, err := FindConfigsByField(ctx, resourceSelector.FieldSelector, selectorOpts...)
			if err != nil {
				return nil, fmt.Errorf("error getting configs with field selectors[%s]: %w", resourceSelector.FieldSelector, err)
			}
			uniqueConfigs = configs
		}

		ids := lo.Map(uniqueConfigs, func(c models.ConfigItem, _ int) string { return c.ID.String() })
		if len(ids) == 0 {
			cacheToUse.Set(hash, ids, time.Minute) // if results weren't found cache it shortly even on the immutable cache
		} else {
			cacheToUse.SetDefault(hash, ids)
		}

		allConfigs = append(allConfigs, uniqueConfigs...)
	}

	return lo.UniqBy(allConfigs, func(c models.ConfigItem) string { return c.ID.String() }), nil
}

// firstResourceSelectorQuery returns an ANDed query from all the fields except the
// label selectors & field selectors.
func firstResourceSelectorQuery(ctx DBContext, table string, resourceSelector types.ResourceSelector) *gorm.DB {
	if resourceSelector.Name == "" && resourceSelector.Namespace == "" && resourceSelector.Agent == "" && len(resourceSelector.Types) == 0 && len(resourceSelector.Statuses) == 0 {
		return nil
	}

	query := ctx.DB()
	if resourceSelector.ID != "" {
		query = query.Where(fmt.Sprintf("%s.id = ?", table), resourceSelector.ID)
	}
	if resourceSelector.Name != "" {
		query = query.Where(fmt.Sprintf("%s.name = ?", table), resourceSelector.Name)
	}
	if resourceSelector.Namespace != "" {
		query = query.Where(fmt.Sprintf("%s.namespace = ?", table), resourceSelector.Namespace)
	}
	if len(resourceSelector.Types) != 0 {
		query = query.Where(fmt.Sprintf("%s.type IN ?", table), resourceSelector.Types)
	}
	if len(resourceSelector.Statuses) != 0 {
		query = query.Where(fmt.Sprintf("%s.status IN ?", table), resourceSelector.Statuses)
	}

	if resourceSelector.Agent != "" {
		if resourceSelector.Agent == "self" {
			query = query.Where(fmt.Sprintf("%s.agent_id = ?", table), uuid.Nil)
		} else if uid, err := uuid.Parse(resourceSelector.Agent); err == nil {
			query = query.Where(fmt.Sprintf("%s.agent_id = ?", table), uid)
		} else { // assume it's an agent name
			query = query.Joins(fmt.Sprintf("LEFT JOIN agents ON %s.agent_id = agents.id", table)).Where("agents.name = ?", resourceSelector.Agent).Where(fmt.Sprintf("%s.agent_id = ?", table), uid)
		}
	}

	return query
}

// LabelledTable is a table that has labels column.
type LabelledTable interface {
	Key() string
	LabelsColumn() string
}

// findResourcesByLabels finds the records of the given table using the label selector.
func findResourcesByLabels[T LabelledTable](ctx context.Context, labelSelector string, opts ...FindOption) ([]T, error) {
	if labelSelector == "" {
		return nil, nil
	}

	var items = make(map[string]T)
	matchLabels := collections.SelectorToMap(labelSelector)
	var tags = make(map[string]string)
	var onlyKeys []string
	for k, v := range matchLabels {
		if v != "" {
			tags[k] = v
		} else {
			onlyKeys = append(onlyKeys, k)
		}
	}

	var anon T
	var configs []T
	if err := apply(ctx.DB().Where(LocalFilter).
		Where(fmt.Sprintf("%s @> ?", anon.LabelsColumn()), types.JSONStringMap(tags)), opts...).
		Find(&configs).Error; err != nil {
		return nil, err
	}
	for _, c := range configs {
		items[c.Key()] = c
	}

	for _, k := range onlyKeys {
		var configs []T
		if err := apply(ctx.DB().Where(LocalFilter).
			Where(fmt.Sprintf("%s ?? ?", anon.LabelsColumn()), k), opts...).
			Find(&configs).Error; err != nil {
			return nil, err
		}

		for _, c := range configs {
			items[c.Key()] = c
		}
	}

	return lo.Values(items), nil
}

func FindComponentsByField(ctx context.Context, fieldSelector string, opts ...FindOption) ([]models.Component, error) {
	if fieldSelector == "" {
		return nil, nil
	}

	matchLabels := collections.SelectorToMap(fieldSelector)
	allowedColumnsAsFields := []string{"topology_type", "owner"}

	columnWhereClauses := map[string]string{}

	var props models.Properties
	for k, v := range matchLabels {
		if collections.Contains(allowedColumnsAsFields, k) {
			columnWhereClauses[k] = v
		} else {
			props = append(props, &models.Property{Name: k, Text: v})
		}
	}

	// If 0 clauses then do not fire query
	if len(columnWhereClauses) == 0 && len(props) == 0 {
		return nil, nil
	}

	query := ctx.DB()
	if len(columnWhereClauses) > 0 {
		query = query.Where(columnWhereClauses)
	}
	if len(props) > 0 {
		query = query.Where("properties @> ?", props)
	}
	var components []models.Component
	if err := apply(query, opts...).
		Find(&components).Error; err != nil {
		return nil, fmt.Errorf("error querying components by fieldSelector[%s]: %w", fieldSelector, err)
	}

	return components, nil
}

func FindConfigsByField(ctx context.Context, fieldSelector string, opts ...FindOption) ([]models.ConfigItem, error) {
	if fieldSelector == "" {
		return nil, nil
	}

	matchLabels := collections.SelectorToMap(fieldSelector)
	allowedColumnsAsFields := []string{"config_class"}

	columnWhereClauses := map[string]string{}

	var props models.Properties
	for k, v := range matchLabels {
		if collections.Contains(allowedColumnsAsFields, k) {
			columnWhereClauses[k] = v
		} else {
			props = append(props, &models.Property{Name: k, Text: v})
		}
	}

	// If 0 clauses then do not fire query
	if len(columnWhereClauses) == 0 && len(props) == 0 {
		return nil, nil
	}

	query := ctx.DB()
	if len(columnWhereClauses) > 0 {
		query = query.Where(columnWhereClauses)
	}
	if len(props) > 0 {
		query = query.Where("properties @> ?", props)
	}
	var configs []models.ConfigItem
	if err := apply(query, opts...).
		Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("error querying components by fieldSelector[%s]: %w", fieldSelector, err)
	}

	return configs, nil
}

func FindChecksByIDs(ctx DBContext, ids []string, opts ...FindOption) ([]models.Check, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var checks []models.Check
	err := apply(ctx.DB().Where(LocalFilter).Where("id IN ?", ids), opts...).Find(&checks).Error
	return checks, err
}

func FindComponentsByIDs(ctx DBContext, ids []string, opts ...FindOption) ([]models.Component, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var components []models.Component
	err := apply(ctx.DB().Where(LocalFilter).Where("id IN ?", ids), opts...).Find(&components).Error
	return components, err
}

func FindConfigsByIDs(ctx DBContext, ids []string, opts ...FindOption) ([]models.ConfigItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var configs []models.ConfigItem
	err := apply(ctx.DB().Where(LocalFilter).Where("id IN ?", ids), opts...).Find(&configs).Error
	return configs, err
}

func FindComponentsByName(ctx context.Context, name string, opts ...FindOption) ([]models.Component, error) {
	if name == "" {
		return nil, nil
	}

	var comps []models.Component
	tx := apply(ctx.DB().Where(LocalFilter).Where("name = ?", name), opts...)
	if err := tx.Find(&comps).Error; err != nil {
		return nil, err
	}

	return comps, nil
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
	return findEntityByField[T](ctx, "id", id)
}

func findEntityByField[T any](ctx DBContext, field, key string, opts ...GetterOption) (*T, error) {
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
