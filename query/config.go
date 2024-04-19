package query

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	extraClausePlugin "github.com/WinterYukky/gorm-extra-clause-plugin"
	"github.com/WinterYukky/gorm-extra-clause-plugin/exclause"
	"github.com/flanksource/commons/duration"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type ConfigSummaryRequestChanges struct {
	Since       string `json:"since"`
	sinceParsed time.Duration
}

type ConfigSummaryRequestAnalysis struct {
	Since       string `json:"since"`
	sinceParsed time.Duration
}

type ConfigSummaryRequest struct {
	Changes  ConfigSummaryRequestChanges  `json:"changes"`
	Analysis ConfigSummaryRequestAnalysis `json:"analysis"`
	Cost     string                       `json:"string"`
	Deleted  bool                         `json:"deleted"`
	Filter   map[string]string            `json:"filter"` // Filter by labels
	GroupBy  []string                     `json:"groupBy"`
	Tags     []string                     `json:"tags"`
}

func (t *ConfigSummaryRequest) OrderBy() string {
	var output []string
	for i := 0; i < len(t.GroupBy); i++ {
		output = append(output, fmt.Sprintf("%d", i+1))
	}
	return strings.Join(output, ", ")
}

func (t *ConfigSummaryRequest) analysisJoin() string {
	output := "LEFT JOIN aggregated_analysis_count ON "
	var clauses []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			clauses = append(clauses, "aggregated_analysis_count .type = config_items.type")
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

func (t *ConfigSummaryRequest) plainSelectClause(appendSelect ...string) []string {
	return append(t.GroupBy, appendSelect...)
}

func (t *ConfigSummaryRequest) selectClause(appendSelect ...string) []string {
	var output []string
	for _, g := range t.GroupBy {
		switch g {
		case "type":
			output = append(output, "config_items.type")
		default:
			output = append(output, fmt.Sprintf("config_items.tags->>'%s' as %s", g, g))
		}
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

	return output
}

func (t *ConfigSummaryRequest) SetDefaults() {
	if t.Changes.Since == "" {
		t.Changes.Since = "2d"
	}

	if t.Analysis.Since == "" {
		t.Analysis.Since = "2d"
	}

	if t.Cost == "" {
		t.Cost = "30d"
	}

	if len(t.GroupBy) == 0 {
		t.GroupBy = []string{"type"}
	}
}

func (t *ConfigSummaryRequest) Parse() error {
	if val, err := duration.ParseDuration(t.Changes.Since); err != nil {
		return fmt.Errorf("changes since is invalid: %w", err)
	} else {
		t.Changes.sinceParsed = time.Duration(val)
	}

	if val, err := duration.ParseDuration(t.Analysis.Since); err != nil {
		return fmt.Errorf("analysis since is invalid: %w", err)
	} else {
		t.Analysis.sinceParsed = time.Duration(val)
	}

	return nil
}

func ConfigSummary(ctx context.Context, req ConfigSummaryRequest) (types.JSON, error) {
	req.SetDefaults()
	if err := req.Parse(); err != nil {
		return nil, api.Errorf(api.EINVALID, err.Error())
	}

	ctx.DB().Use(extraClausePlugin.New())

	groupBy := strings.Join(req.groupBy(), ",")

	changesGrouped := exclause.NewWith(
		"changes_grouped",
		ctx.DB().Select(req.selectClause("COUNT(*) AS count")).
			Model(&models.ConfigChange{}).
			Joins("LEFT JOIN config_items ON config_changes.config_id = config_items.id").
			Where("config_items.deleted_at IS NULL").
			Where("NOW() - config_changes.created_at <= ?", req.Changes.sinceParsed).
			Group(groupBy),
	)

	analysisGrouped := exclause.NewWith(
		"analysis_grouped",
		ctx.DB().Select(req.selectClause("config_analysis.analysis_type", "COUNT(*) AS count")).
			Model(&models.ConfigAnalysis{}).
			Joins("LEFT JOIN config_items ON config_analysis.config_id = config_items.id").
			Where("config_items.deleted_at IS NULL").
			Where("NOW() - config_analysis.first_observed <= ?", req.Analysis.sinceParsed).
			Group(groupBy).Group("config_analysis.analysis_type"),
	)

	aggregatedAnalysisGrouped := exclause.NewWith(
		"aggregated_analysis_count",
		ctx.DB().Select(req.plainSelectClause("json_object_agg(analysis_type, count)::jsonb AS analysis")).
			Table("analysis_grouped").
			Group(strings.Join(req.GroupBy, ",")),
	)

	final := exclause.NewWith(
		"summary",
		ctx.DB().
			Select(req.selectClause("changes_grouped.count AS changes", "aggregated_analysis_count.analysis AS analysis", "COUNT(*) AS total_configs")).
			Model(&models.ConfigItem{}).
			Joins(req.changesJoin()).
			Joins(req.analysisJoin()).
			Where("config_items.deleted_at IS NULL").
			Group(groupBy).
			Group("changes_grouped.count, aggregated_analysis_count.analysis").
			Order(req.OrderBy()),
	)

	var res []types.JSON
	if err := ctx.DB().
		Debug().
		Clauses(changesGrouped).
		Clauses(analysisGrouped).
		Clauses(aggregatedAnalysisGrouped).
		Clauses(final).
		Select("json_agg(row_to_json(summary))").
		Table("summary").Scan(&res).Error; err != nil {
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

func FindConfig(ctx context.Context, query types.ConfigQuery) (*models.ConfigItem, error) {
	res, err := FindConfigsByResourceSelector(ctx, query.ToResourceSelector())
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return &res[0], nil
}

func FindConfigs(ctx context.Context, config types.ConfigQuery) ([]models.ConfigItem, error) {
	return FindConfigsByResourceSelector(ctx, config.ToResourceSelector())
}

func FindConfigIDs(ctx context.Context, config types.ConfigQuery) ([]uuid.UUID, error) {
	return FindConfigIDsByResourceSelector(ctx, config.ToResourceSelector())
}

func FindConfigsByResourceSelector(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]models.ConfigItem, error) {
	items, err := FindConfigIDsByResourceSelector(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigsByIDs(ctx, items)
}

func FindConfigIDsByResourceSelector(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allConfigs []uuid.UUID

	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector(ctx, resourceSelector, "config_items", models.AllowedColumnFieldsInConfigs)
		if err != nil {
			return nil, err
		}

		allConfigs = append(allConfigs, items...)
	}

	return allConfigs, nil
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
