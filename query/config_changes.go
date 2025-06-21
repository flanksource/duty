package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/pkg/kube/labels"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/timberio/go-datemath"
	"gorm.io/gorm/clause"
)

type ChangeRelationDirection string

const (
	CatalogChangeRecursiveUpstream   ChangeRelationDirection = "upstream"
	CatalogChangeRecursiveDownstream ChangeRelationDirection = "downstream"
	CatalogChangeRecursiveNone       ChangeRelationDirection = "none"
	CatalogChangeRecursiveAll        ChangeRelationDirection = "all"
)

var allRecursiveOptions = []ChangeRelationDirection{CatalogChangeRecursiveUpstream, CatalogChangeRecursiveDownstream, CatalogChangeRecursiveNone, CatalogChangeRecursiveAll}

var allowedConfigChangesSortColumns = []string{"name", "change_type", "summary", "source", "created_at", "count"}

type CatalogChangesSearchRequest struct {
	CatalogID             string `query:"id" json:"id"`
	ConfigType            string `query:"config_type" json:"config_type"`
	ChangeType            string `query:"type" json:"type"`
	Severity              string `query:"severity" json:"severity"`
	IncludeDeletedConfigs bool   `query:"include_deleted_configs" json:"include_deleted_configs"`
	Depth                 int    `query:"depth" json:"depth"`
	CreatedByRaw          string `query:"created_by" json:"created_by"`
	Summary               string `query:"summary" json:"summary"`
	Source                string `query:"source" json:"source"`
	Tags                  string `query:"tags" json:"tags"`

	// To Fetch from a particular agent, provide the agent id.
	// Use `local` keyword to filter by the local agent.
	AgentID string `query:"agent_id" json:"agent_id"`

	createdBy         *uuid.UUID
	externalCreatedBy string

	// From date in datemath format
	From string `query:"from" json:"from"`
	// To date in datemath format
	To string `query:"to" json:"to"`

	PageSize  int    `query:"page_size" json:"page_size"`
	Page      int    `query:"page" json:"page"`
	SortBy    string `query:"sort_by" json:"sort_by"`
	sortOrder string

	// upstream | downstream | both
	Recursive ChangeRelationDirection `query:"recursive" json:"recursive"`

	// FIXME: Soft toggle does not work with Recursive=both
	// In that case, soft relations are always returned
	// It also returns ALL soft relations throughout the tree
	// not just soft related to the main config item
	Soft bool `query:"soft" json:"soft"`

	fromParsed time.Time
	toParsed   time.Time
}

func (t CatalogChangesSearchRequest) String() string {
	s := ""
	if t.AgentID != "" {
		s += fmt.Sprintf("agent: %s", t.AgentID)
	}
	if t.CatalogID != "" {
		s += fmt.Sprintf("id: %s ", t.CatalogID)
	}
	if t.ConfigType != "" {
		s += fmt.Sprintf("config_type: %s ", t.ConfigType)
	}
	if t.ChangeType != "" {
		s += fmt.Sprintf("type: %s ", t.ChangeType)
	}
	if t.Severity != "" {
		s += fmt.Sprintf("severity: %s ", t.Severity)
	}
	if t.Source != "" {
		s += fmt.Sprintf("source: %s ", t.Source)
	}
	if t.IncludeDeletedConfigs {
		s += fmt.Sprintf("include_deleted_configs: %t ", t.IncludeDeletedConfigs)
	}
	if t.Depth != 0 {
		s += fmt.Sprintf("depth: %d ", t.Depth)
	}
	if t.CreatedByRaw != "" {
		s += fmt.Sprintf("created_by: %s ", t.CreatedByRaw)
	}
	if t.Summary != "" {
		s += fmt.Sprintf("summary: %s ", t.Summary)
	}
	if t.Tags != "" {
		s += fmt.Sprintf("tags: %s ", t.Tags)
	}
	if t.From != "" {
		s += fmt.Sprintf("from: %s ", t.From)
	}
	if t.To != "" {
		s += fmt.Sprintf("to: %s ", t.To)
	}
	if t.PageSize != 0 {
		s += fmt.Sprintf("page_size: %d ", t.PageSize)
	}
	if t.Page != 0 {
		s += fmt.Sprintf("page: %d ", t.Page)
	}
	if t.SortBy != "" {
		s += fmt.Sprintf("sort_by: %s %s ", t.SortBy, t.sortOrder)
	}
	if t.Recursive != "" {
		s += fmt.Sprintf("recursive: %s ", t.Recursive)
	}
	return s
}

func (t *CatalogChangesSearchRequest) SetDefaults() {
	if t.PageSize <= 0 {
		t.PageSize = 50
	}

	if t.Page <= 0 {
		t.Page = 1
	}

	if t.From == "" && t.To == "" {
		t.From = "now-2d"
	}

	if t.Recursive == "" {
		t.Recursive = CatalogChangeRecursiveDownstream
	}

	if t.Depth <= 0 {
		t.Depth = 5
	}

	if t.AgentID == "local" {
		t.AgentID = uuid.Nil.String()
	}
}

func (t *CatalogChangesSearchRequest) Validate() error {
	if !lo.Contains(allRecursiveOptions, t.Recursive) {
		return fmt.Errorf("'recursive' must be one of %v", allRecursiveOptions)
	}

	if t.From != "" {
		if expr, err := datemath.Parse(t.From); err != nil {
			return fmt.Errorf("invalid 'from' param: %w", err)
		} else {
			t.fromParsed = expr.Time()
		}
	}

	if t.To != "" {
		if expr, err := datemath.Parse(t.To); err != nil {
			return fmt.Errorf("invalid 'to' param: %w", err)
		} else {
			t.toParsed = expr.Time()
		}
	}

	if !t.fromParsed.IsZero() && !t.toParsed.IsZero() && !t.fromParsed.Before(t.toParsed) {
		return fmt.Errorf("'from' must be before 'to'")
	}

	if t.SortBy != "" {
		if strings.HasPrefix(t.SortBy, "-") {
			t.sortOrder = "desc"
			t.SortBy = strings.TrimPrefix(t.SortBy, "-")
		}

		if !lo.Contains(allowedConfigChangesSortColumns, t.SortBy) {
			return fmt.Errorf("invalid 'sort_by' param: %s. allowed sort fields are: %s", t.SortBy, strings.Join(allowedConfigChangesSortColumns, ", "))
		}
	}

	if t.CreatedByRaw != "" {
		if u, err := uuid.Parse(t.CreatedByRaw); err == nil {
			t.createdBy = &u
		} else {
			t.externalCreatedBy = t.CreatedByRaw
		}
	}

	if t.AgentID != "" {
		if _, err := uuid.Parse(t.AgentID); err != nil {
			return fmt.Errorf("agent_id(%s) must either be a valid uuid or `local`", t.AgentID)
		}
	}

	return nil
}

type ConfigChangeRow struct {
	AgentID           string              `gorm:"column:agent_id" json:"agent_id"`
	ExternalChangeId  string              `gorm:"column:external_change_id" json:"external_change_id"`
	ID                string              `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	ConfigID          string              `gorm:"column:config_id;default:''" json:"config_id"`
	DeletedAt         *time.Time          `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	ChangeType        string              `gorm:"column:change_type" json:"change_type" faker:"oneof:  RunInstances, diff"`
	Severity          string              `gorm:"column:severity" json:"severity"  faker:"oneof: critical, high, medium, low, info"`
	Source            string              `gorm:"column:source" json:"source"`
	Summary           string              `gorm:"column:summary;default:null" json:"summary,omitempty"`
	CreatedAt         *time.Time          `gorm:"column:created_at" json:"created_at"`
	Count             int                 `gorm:"column:count" json:"count"`
	FirstObserved     *time.Time          `gorm:"column:first_observed" json:"first_observed,omitempty"`
	ConfigName        string              `gorm:"column:name" json:"name,omitempty"`
	ConfigType        string              `gorm:"column:type" json:"type,omitempty"`
	Tags              types.JSONStringMap `gorm:"column:tags" json:"tags,omitempty"`
	CreatedBy         *uuid.UUID          `gorm:"column:created_by" json:"created_by,omitempty"`
	ExternalCreatedBy string              `gorm:"column:external_created_by" json:"external_created_by,omitempty"`
}

type CatalogChangesSearchResponse struct {
	Summary map[string]int    `json:"summary,omitempty"`
	Total   int64             `json:"total,omitempty"`
	Changes []ConfigChangeRow `json:"changes,omitempty"`
}

func (t *CatalogChangesSearchResponse) Summarize() {
	t.Summary = make(map[string]int)
	for _, c := range t.Changes {
		t.Summary[c.ChangeType]++
	}
}

func formSeverityQuery(severity string) string {
	if strings.HasPrefix(severity, "!") {
		// For `Not` queries, we don't need to make any changes.
		return severity
	}

	severities := []models.Severity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	}

	var applicable []string
	for _, s := range severities {
		applicable = append(applicable, string(s))
		if string(s) == severity {
			break
		}
	}

	return strings.Join(applicable, ",")
}

func FindCatalogChanges(ctx context.Context, req CatalogChangesSearchRequest) (*CatalogChangesSearchResponse, error) {
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, api.Errorf(api.EINVALID, "bad request: %v", err)
	}
	ctx.Tracef("query changes: %s", req)

	var clauses []clause.Expression

	query := ctx.DB()

	if req.AgentID != "" {
		clause, err := parseAndBuildFilteringQuery(req.AgentID, "agent_id", false)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause...)
	}

	if req.ConfigType != "" {
		clause, err := parseAndBuildFilteringQuery(req.ConfigType, "type", false)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause...)
	}

	if req.ChangeType != "" {
		clause, err := parseAndBuildFilteringQuery(req.ChangeType, "change_type", false)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause...)
	}

	if req.Severity != "" {
		clause, err := parseAndBuildFilteringQuery(formSeverityQuery(req.Severity), "severity", false)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse severity: %v", err)
		}
		clauses = append(clauses, clause...)
	}

	if req.Summary != "" {
		clause, err := parseAndBuildFilteringQuery(req.Summary, "summary", true)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse summary: %v", err)
		}
		clauses = append(clauses, clause...)
	}

	if req.Source != "" {
		clause, err := parseAndBuildFilteringQuery(req.Source, "source", true)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse source: %v", err)
		}
		clauses = append(clauses, clause...)
	}

	if req.Tags != "" {
		parsedLabelSelector, err := labels.Parse(req.Tags)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse label selector: %v", err)
		}
		requirements, _ := parsedLabelSelector.Requirements()
		for _, r := range requirements {
			query = jsonColumnRequirementsToSQLClause(query, "tags", r)
		}
	}

	if !req.fromParsed.IsZero() {
		clauses = append(clauses, clause.Gte{Column: clause.Column{Name: "created_at"}, Value: req.fromParsed})
	}

	if !req.toParsed.IsZero() {
		clauses = append(clauses, clause.Lte{Column: clause.Column{Name: "created_at"}, Value: req.toParsed})
	}

	if req.createdBy != nil {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: "created_by"}, Value: req.createdBy})
	}

	if req.externalCreatedBy != "" {
		clause, err := parseAndBuildFilteringQuery(req.externalCreatedBy, "external_created_by", true)
		if err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to parse external createdby: %v", err)
		}
		clauses = append(clauses, clause...)
	}

	if !req.IncludeDeletedConfigs {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: "deleted_at"}, Value: nil})
	}

	table := query.Table("catalog_changes")
	if err := uuid.Validate(req.CatalogID); err == nil {
		table = query.Table("related_changes_recursive(?,?,?,?,?)", req.CatalogID, req.Recursive, req.IncludeDeletedConfigs, req.Depth, req.Soft)
	} else {
		clause, err := parseAndBuildFilteringQuery(req.CatalogID, "config_id", false)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause...)
	}

	var output CatalogChangesSearchResponse
	if err := table.Clauses(clauses...).Count(&output.Total).Error; err != nil {
		return nil, err
	}

	if output.Total == 0 {
		return &output, nil
	}

	if req.SortBy != "" {
		clauses = append(clauses,
			clause.OrderBy{Columns: []clause.OrderByColumn{
				{
					Column: clause.Column{Name: req.SortBy},
					Desc:   req.sortOrder == "desc",
				},
			}})
	}

	clauses = append(clauses,
		clause.Limit{Limit: lo.ToPtr(req.PageSize), Offset: (req.Page - 1) * req.PageSize},
	)

	if err := table.Clauses(clauses...).Find(&output.Changes).Error; err != nil {
		return nil, err
	}

	output.Summarize()
	return &output, nil
}

func FindConfigChangesByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]models.CatalogChange, error) {
	items, err := FindConfigChangeIDsByResourceSelector(ctx, limit, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetCatalogChangesByIDs(ctx, items)
}

func FindConfigChangeIDsByResourceSelector(ctx context.Context, limit int, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	return queryTableWithResourceSelectors(ctx, "catalog_changes", limit, resourceSelectors...)
}

func GetCatalogChangesByIDs(ctx context.Context, ids []uuid.UUID) ([]models.CatalogChange, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var catalogChanges []models.CatalogChange
	if err := ctx.DB().Table("catalog_changes").Where("id IN ?", ids).Find(&catalogChanges).Error; err != nil {
		return nil, err
	}

	return catalogChanges, nil
}
