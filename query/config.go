package query

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/WinterYukky/gorm-extra-clause-plugin/exclause"
	"github.com/flanksource/commons/duration"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query/grammar"
	"github.com/flanksource/duty/types"
)

type ConfigSummaryRequestChanges struct {
	Since       string `json:"since"`
	sinceParsed time.Duration
}

type ConfigSummaryRequestAnalysis struct{}

type ConfigSummaryRequest struct {
	Changes  ConfigSummaryRequestChanges  `json:"changes"`
	Analysis ConfigSummaryRequestAnalysis `json:"analysis"`
	Cost     string                       `json:"cost"`
	Deleted  bool                         `json:"deleted"`
	Filter   map[string]string            `json:"filter"` // Filter by labels
	GroupBy  []string                     `json:"groupBy"`
	Tags     []string                     `json:"tags"`
	Status   string                       `json:"status"`
	Health   string                       `json:"health"`
}

func (t *ConfigSummaryRequest) OrderBy() string {
	var output []string
	for i := 0; i < len(t.GroupBy); i++ {
		output = append(output, fmt.Sprintf("%d", i+1))
	}
	return strings.Join(output, ", ")
}

func (t *ConfigSummaryRequest) healthSummaryJoin() string {
	output := "LEFT JOIN aggregated_health_count ON "
	var clauses []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			clauses = append(clauses, "aggregated_health_count.type = config_items.type")
		default:
			clauses = append(clauses, fmt.Sprintf("aggregated_health_count.%s = config_items.tags->>'%s'", g, g))
		}
	}

	return output + strings.Join(clauses, " AND ")
}

func (t *ConfigSummaryRequest) analysisJoin() string {
	output := "LEFT JOIN aggregated_analysis_count ON "
	var clauses []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			clauses = append(clauses, "aggregated_analysis_count.type = config_items.type")
		default:
			clauses = append(clauses, fmt.Sprintf("aggregated_analysis_count.%s = config_items.tags->>'%s'", g, g))
		}
	}

	return output + strings.Join(clauses, " AND ")
}

func (t *ConfigSummaryRequest) changesJoin() string {
	output := "LEFT JOIN changes_grouped ON "
	var clauses []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			clauses = append(clauses, "changes_grouped.type = config_items.type")
		default:
			clauses = append(clauses, fmt.Sprintf("changes_grouped.%s = config_items.tags->>'%s'", g, g))
		}
	}

	return output + strings.Join(clauses, " AND ")
}

func (t *ConfigSummaryRequest) changesAnalysisJoin() string {
	output := "LEFT JOIN aggregated_analysis ON "
	var clauses []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			clauses = append(clauses, "aggregated_analysis.type = config_items.type")
		default:
			clauses = append(clauses, fmt.Sprintf("aggregated_analysis.%s = config_items.tags->>'%s'", g, g))
		}
	}

	return output + strings.Join(clauses, " AND ")
}

func (t ConfigSummaryRequest) plainSelectClause(appendSelect ...string) []string {
	output := make([]string, len(t.GroupBy)+len(appendSelect))
	copy(output, t.GroupBy)
	copy(output[len(t.GroupBy):], appendSelect)
	return output
}

func (t *ConfigSummaryRequest) summarySelectClause() []string {
	cols := []string{
		"aggregated_health_count.health AS health",
		"MAX(config_items.created_at) AS created_at",
		"MAX(config_items.updated_at) AS updated_at",
		"COUNT(*) AS count",
	}

	if t.Cost != "" {
		cols = append(cols, fmt.Sprintf("SUM(cost_total_%s) as cost_%s", t.Cost, t.Cost))
	}

	if slices.Contains([]string{"3d", "7d", "30d"}, t.Changes.Since) {
		cols = append(cols, fmt.Sprintf("COALESCE(sum(config_item_summary_%s.config_changes_count), 0) AS changes, COALESCE(aggregated_analysis.total_analysis, '{}'::jsonb) AS analysis", t.Changes.Since))
	} else {
		if t.Changes.Since != "" {
			cols = append(cols, "changes_grouped.count AS changes")
		}
		cols = append(cols,
			"aggregated_analysis_count.analysis AS analysis",
		)
	}

	return t.baseSelectClause(cols...)
}

func (t *ConfigSummaryRequest) baseSelectClause(appendSelect ...string) []string {
	var output []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			output = append(output, "config_items.type")
		default:
			output = append(output, fmt.Sprintf("config_items.tags->>'%s' as %s", g, g))
		}
	}

	for _, tag := range t.Tags {
		output = append(output, fmt.Sprintf("config_items.tags->>'%s' as %s", tag, tag))
	}

	if len(output) == 0 {
		output = []string{"config_items.type"}
	}

	output = append(output, appendSelect...)
	return output
}

func (t *ConfigSummaryRequest) groupBy() []string {
	var output []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			output = append(output, "config_items.type")
		default:
			output = append(output, fmt.Sprintf("config_items.tags->>'%s'", g))
		}
	}

	for _, tag := range t.Tags {
		output = append(output, fmt.Sprintf("config_items.tags->>'%s'", tag))
	}

	return output
}

func (t *ConfigSummaryRequest) SetDefaults() {
	if len(t.GroupBy) == 0 {
		t.GroupBy = []string{"type"}
	}
}

func (t *ConfigSummaryRequest) Parse() error {
	if t.Changes.Since != "" {
		if val, err := duration.ParseDuration(t.Changes.Since); err != nil {
			return fmt.Errorf("changes since is invalid: %w", err)
		} else {
			t.Changes.sinceParsed = time.Duration(val)
		}
	}

	switch t.Cost {
	case "1d", "7d", "30d", "":
		// do nothing
	default:
		return fmt.Errorf("cost range is not allowed. allowed (1d, 7d, 30d)")
	}

	return nil
}

func (t ConfigSummaryRequest) configDeleteClause() string {
	if !t.Deleted {
		return "config_items.deleted_at IS NULL"
	}

	return ""
}

func (t ConfigSummaryRequest) statusClause() []clause.Expression {
	clause, _ := parseAndBuildFilteringQuery(t.Status, "config_items.status", false)
	return clause
}

func (t ConfigSummaryRequest) healthClause() []clause.Expression {
	clause, _ := parseAndBuildFilteringQuery(t.Health, "config_items.health", false)
	return clause
}

func (t *ConfigSummaryRequest) filterClause(q *gorm.DB) *gorm.DB {
	var includeClause *gorm.DB
	var excludeClause *gorm.DB

	for k, v := range t.Filter {
		query, _ := grammar.ParseFilteringQueryV2(v, true)

		if len(query.Not.In) > 0 {
			if excludeClause == nil {
				excludeClause = q
			}

			for _, excludeValue := range query.Not.In {
				excludeClause = excludeClause.Where("NOT (config_items.labels @> ?)", types.JSONStringMap{k: excludeValue.(string)})
			}
		}

		if len(query.In) > 0 {
			if includeClause == nil {
				includeClause = q
			}

			for _, includeValue := range query.In {
				includeClause = includeClause.Or("config_items.labels @> ?", types.JSONStringMap{k: includeValue.(string)})
			}
		}
	}

	if includeClause != nil {
		q = q.Where(includeClause)
	}

	if excludeClause != nil {
		q = q.Where(excludeClause)
	}

	return q
}

func ConfigSummary(ctx context.Context, req ConfigSummaryRequest) (types.JSON, error) {
	req.SetDefaults()
	if err := req.Parse(); err != nil {
		return nil, api.Errorf(api.EINVALID, "%s", err)
	}

	groupBy := strings.Join(req.groupBy(), ",")
	plainGroupBy := strings.Join(req.GroupBy, ",")

	healthGrouped := exclause.NewWith(
		"health_grouped",
		ctx.DB().Select(req.baseSelectClause("health, COUNT(health) AS count")).
			Model(&models.ConfigItem{}).
			Where("health IS NOT NULL").
			Where(req.configDeleteClause()).
			Where(req.filterClause(ctx.DB())).
			Group(groupBy).
			Group("health"),
	)

	healthAggregated := exclause.NewWith(
		"aggregated_health_count",
		ctx.DB().Select(req.plainSelectClause("jsonb_object_agg(health_grouped.health, count)::jsonb AS health")).
			Table("health_grouped").
			Group(plainGroupBy),
	)

	// Keep track of all the ctes in this query (in order)
	withClauses := []clause.Expression{healthGrouped, healthAggregated}

	summaryQuery := ctx.DB().
		Select(req.summarySelectClause()).
		Model(&models.ConfigItem{}).
		Joins(req.healthSummaryJoin()).
		Where(req.configDeleteClause()).
		Where(req.filterClause(ctx.DB())).
		Clauses(req.statusClause()...).
		Clauses(req.healthClause()...).
		Group(groupBy).
		Group("aggregated_health_count.health").
		Order(req.OrderBy())

	if slices.Contains([]string{"3d", "7d", "30d"}, req.Changes.Since) {
		tableName := fmt.Sprintf("config_item_summary_%s", req.Changes.Since)
		changesAnalysisGrouped := exclause.NewWith(
			"changes_analysis_grouped",
			ctx.DB().Select(req.baseSelectClause(fmt.Sprintf("SUM(%s.config_changes_count) AS total_changes, COALESCE(kv_pair.key, '') AS key, SUM((kv_pair.value::int)) AS value_sum", tableName))).
				Table(tableName).
				Joins(fmt.Sprintf("LEFT JOIN config_items ON %s.config_id = config_items.id", tableName)).
				Joins(fmt.Sprintf("LEFT JOIN jsonb_each_text(%s.config_analysis_type_counts) AS kv_pair(key, value) ON %s.config_analysis_type_counts IS NOT NULL", tableName, tableName)).
				Where(req.configDeleteClause()).
				Where("kv_pair.key IS NOT NULL AND kv_pair.key <> ''").
				Where(req.filterClause(ctx.DB())).
				Group(groupBy).Group("kv_pair.key"),
		)

		aggregatedAnalysis := exclause.NewWith(
			"aggregated_analysis",
			ctx.DB().Select(req.plainSelectClause("COALESCE(jsonb_object_agg(key, value_sum), '{}'::jsonb) AS total_analysis")).
				Table("changes_analysis_grouped").
				Group(plainGroupBy),
		)

		withClauses = append(withClauses, changesAnalysisGrouped, aggregatedAnalysis)
		summaryQuery = summaryQuery.
			Joins(req.changesAnalysisJoin()).
			Joins(fmt.Sprintf("LEFT JOIN %s ON %s.config_id = config_items.id", tableName, tableName)).
			Group("aggregated_analysis.total_analysis")

	} else {
		if req.Changes.Since != "" {
			changesGrouped := exclause.NewWith(
				"changes_grouped",
				ctx.DB().Select(req.baseSelectClause("COUNT(*) AS count")).
					Model(&models.ConfigChange{}).
					Joins("LEFT JOIN config_items ON config_changes.config_id = config_items.id").
					Where(req.configDeleteClause()).
					Where(req.filterClause(ctx.DB())).
					Where("NOW() - config_changes.created_at <= ?", req.Changes.sinceParsed).
					Group(groupBy),
			)

			summaryQuery = summaryQuery.Joins(req.changesJoin()).Group("changes_grouped.count")
			withClauses = append(withClauses, changesGrouped)
		}

		analysisGrouped := exclause.NewWith(
			"analysis_grouped",
			ctx.DB().Select(req.baseSelectClause("config_analysis.analysis_type", "COUNT(*) AS count")).
				Model(&models.ConfigAnalysis{}).
				Joins("LEFT JOIN config_items ON config_analysis.config_id = config_items.id").
				Where(req.configDeleteClause()).
				Where(req.filterClause(ctx.DB())).
				Where("config_analysis.status = ?", models.AnalysisStatusOpen).
				Group(groupBy).Group("config_analysis.analysis_type"),
		)

		analysisAggregated := exclause.NewWith(
			"aggregated_analysis_count",
			ctx.DB().Select(req.plainSelectClause("json_object_agg(analysis_type, count)::jsonb AS analysis")).
				Table("analysis_grouped").
				Group(plainGroupBy),
		)

		summaryQuery = summaryQuery.Joins(req.analysisJoin()).Group("aggregated_analysis_count.analysis")
		withClauses = append(withClauses, analysisGrouped, analysisAggregated)
	}

	withClauses = append(withClauses, exclause.NewWith("summary", summaryQuery))

	var res []types.JSON
	if err := ctx.DB().Clauses(withClauses...).Select("json_agg(row_to_json(summary))").Table("summary").Scan(&res).Error; err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0], nil
}

func GetConfigsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ConfigItem, error) {
	var configs []models.ConfigItem
	for i := range ids {
		config, err := ConfigItemFromCache(ctx, ids[i].String())
		if err != nil {
			return nil, err
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func GetConfigItemSummaryByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ConfigItemSummary, error) {
	var configs []models.ConfigItemSummary
	for i := range ids {
		config, err := ConfigItemSummaryFromCache(ctx, ids[i].String())
		if err != nil {
			return nil, err
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func FindConfig(ctx context.Context, query types.ConfigQuery) (*models.ConfigItem, error) {
	res, err := FindConfigsByResourceSelector(ctx, -1, query.ToResourceSelector())
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return &res[0], nil
}

func FindConfigs(ctx context.Context, limit int, config types.ConfigQuery) ([]models.ConfigItem, error) {
	return FindConfigsByResourceSelector(ctx, limit, config.ToResourceSelector())
}

func FindConfigIDs(ctx context.Context, limit int, config types.ConfigQuery) ([]uuid.UUID, error) {
	return FindConfigIDsByResourceSelector(ctx, limit, config.ToResourceSelector())
}

func FindConfigsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.ConfigItem, error) {
	items, err := FindConfigIDsByResourceSelector(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigsByIDs(ctx, items)
}

func FindConfigItemSummaryByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.ConfigItemSummary, error) {
	items, err := FindConfigIDsByResourceSelector(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigItemSummaryByIDs(ctx, items)
}

func FindConfigItemSummaryIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, models.ConfigItemSummary{}.TableName(), limit, resourceSelectors...)
}

func FindConfigIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, "config_items", limit, resourceSelectors...)
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

func FindConfigChildrenIDsByLocation(ctx context.Context, configID uuid.UUID, prefix string) ([]uuid.UUID, error) {
	var children []uuid.UUID
	if err := ctx.DB().Raw(`SELECT id FROM get_children_id_by_location(?, ?)`, configID, prefix).Scan(&children).Error; err != nil {
		return nil, err
	}

	return children, nil
}

func FindConfigParentIDsByLocation(ctx context.Context, configID uuid.UUID, prefix string) ([]uuid.UUID, error) {
	var parents []uuid.UUID
	if err := ctx.DB().Raw(`SELECT id FROM get_parent_ids_by_location(?, ?)`, configID, prefix).Scan(&parents).Error; err != nil {
		return nil, err
	}

	return parents, nil
}

type ConfigMinimal struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Type string    `json:"type"`
}

func FindConfigChildrenByLocation(ctx context.Context, configID uuid.UUID, prefix string, includeDeleted bool) ([]ConfigMinimal, error) {
	var children []ConfigMinimal
	if err := ctx.DB().Raw(`SELECT id, name, type FROM get_children_by_location(?, ?, ?)`, configID, prefix, includeDeleted).Scan(&children).Error; err != nil {
		return nil, err
	}

	return children, nil
}

func FindConfigParentsByLocation(ctx context.Context, configID uuid.UUID, prefix string, includeDeleted bool) ([]ConfigMinimal, error) {
	var parents []ConfigMinimal
	if err := ctx.DB().Raw(`SELECT id, name, type FROM get_parents_by_location(?, ?, ?)`, configID, prefix, includeDeleted).Scan(&parents).Error; err != nil {
		return nil, err
	}

	return parents, nil
}
