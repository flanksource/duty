package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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
	BaseCatalogSearch `json:",inline"`

	ChangeType   string `query:"type" json:"type"`
	Severity     string `query:"severity" json:"severity"`
	CreatedByRaw string `query:"created_by" json:"created_by"`
	Summary      string `query:"summary" json:"summary"`
	Source       string `query:"source" json:"source"`

	createdBy         *uuid.UUID
	externalCreatedBy string

	// FromInsertedAt in datemath format
	FromInsertedAt string `query:"from_inserted_at" json:"from_inserted_at"`
	// ToInsertedAt in datemath format
	ToInsertedAt string `query:"to_inserted_at" json:"to_inserted_at"`

	fromInsertedAtParsed time.Time
	toInsertedAtParsed   time.Time
}

func (t CatalogChangesSearchRequest) String() string {
	s := t.BaseCatalogSearch.String()
	if t.ChangeType != "" {
		s += fmt.Sprintf(" type=%s", t.ChangeType)
	}
	if t.Severity != "" {
		s += fmt.Sprintf(" severity=%s", t.Severity)
	}
	if t.Source != "" {
		s += fmt.Sprintf(" source=%s", t.Source)
	}
	if t.CreatedByRaw != "" {
		s += fmt.Sprintf(" created_by=%s", t.CreatedByRaw)
	}
	if t.Summary != "" {
		s += fmt.Sprintf(" summary=%s", t.Summary)
	}
	return s
}

func (t *CatalogChangesSearchRequest) SetDefaults() {
	if t.From == "" && t.To == "" {
		t.From = "now-2d"
	}
	t.BaseCatalogSearch.SetDefaults()
}

func (t *CatalogChangesSearchRequest) Validate() error {
	if err := t.BaseCatalogSearch.Validate(); err != nil {
		return err
	}

	if !lo.Contains(allRecursiveOptions, t.Recursive) {
		return fmt.Errorf("'recursive' must be one of %v", allRecursiveOptions)
	}

	if t.FromInsertedAt != "" {
		if expr, err := datemath.Parse(t.FromInsertedAt); err != nil {
			return fmt.Errorf("invalid 'from_inserted_at' param: %w", err)
		} else {
			t.fromInsertedAtParsed = expr.Time()
		}
	}

	if t.ToInsertedAt != "" {
		if expr, err := datemath.Parse(t.ToInsertedAt); err != nil {
			return fmt.Errorf("invalid 'to_inserted_at' param: %w", err)
		} else {
			t.toInsertedAtParsed = expr.Time()
		}
	}

	if !t.fromInsertedAtParsed.IsZero() && !t.toInsertedAtParsed.IsZero() && !t.fromInsertedAtParsed.Before(t.toInsertedAtParsed) {
		return fmt.Errorf("'from_inserted_at' must be before 'to_inserted_at'")
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
	Path              string              `gorm:"column:path" json:"path,omitempty"`
	InsertedAt        *time.Time          `gorm:"column:inserted_at" json:"inserted_at,omitempty"`
}

func (r ConfigChangeRow) QueryLogSummary() string {
	return r.ChangeType
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
	found := false
	for _, s := range severities {
		applicable = append(applicable, string(s))
		if string(s) == severity {
			found = true
			break
		}
	}

	if !found {
		return "__invalid__"
	}

	return strings.Join(applicable, ",")
}

func FindCatalogChanges(ctx context.Context, req CatalogChangesSearchRequest) (result *CatalogChangesSearchResponse, err error) {
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, api.Errorf(api.EINVALID, "bad request: %v", err)
	}

	timer := NewQueryLogger(ctx).Start("CatalogChanges").Arg("query", req.String())
	defer timer.End(&err)

	configIDs, err := req.ResolveConfigIDs(ctx)
	if err != nil {
		return nil, err
	}

	baseClauses, tagsFn := req.ApplyClauses()
	var clauses []clause.Expression
	clauses = append(clauses, baseClauses...)

	dbQuery := ctx.DB()
	if tagsFn != nil {
		dbQuery = tagsFn(dbQuery)
	}

	if req.ChangeType != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.ChangeType, "change_type", false); parseErr == nil {
			clauses = append(clauses, c...)
		} else {
			return nil, parseErr
		}
	}

	if req.Severity != "" {
		if c, parseErr := parseAndBuildFilteringQuery(formSeverityQuery(req.Severity), "severity", false); parseErr == nil {
			clauses = append(clauses, c...)
		} else {
			return nil, api.Errorf(api.EINVALID, "failed to parse severity: %v", parseErr)
		}
	}

	if req.Summary != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.Summary, "summary", true); parseErr == nil {
			clauses = append(clauses, c...)
		} else {
			return nil, api.Errorf(api.EINVALID, "failed to parse summary: %v", parseErr)
		}
	}

	if req.Source != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.Source, "source", true); parseErr == nil {
			clauses = append(clauses, c...)
		} else {
			return nil, api.Errorf(api.EINVALID, "failed to parse source: %v", parseErr)
		}
	}

	if !req.fromInsertedAtParsed.IsZero() {
		clauses = append(clauses, clause.Gte{Column: clause.Column{Name: "inserted_at"}, Value: req.fromInsertedAtParsed})
	}

	if !req.toInsertedAtParsed.IsZero() {
		clauses = append(clauses, clause.Lte{Column: clause.Column{Name: "inserted_at"}, Value: req.toInsertedAtParsed})
	}

	if req.createdBy != nil {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: "created_by"}, Value: req.createdBy})
	}

	if req.externalCreatedBy != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.externalCreatedBy, "external_created_by", true); parseErr == nil {
			clauses = append(clauses, c...)
		} else {
			return nil, api.Errorf(api.EINVALID, "failed to parse external createdby: %v", parseErr)
		}
	}

	// Determine table: single UUID uses related_changes_recursive, multi-ID or query uses IN clause
	table := dbQuery.Table("catalog_changes")
	if len(configIDs) == 1 {
		table = dbQuery.Table("related_changes_recursive(?,?,?,?,?)", configIDs[0], req.Recursive, req.IncludeDeletedConfigs, req.Depth, req.Soft)
	} else if len(configIDs) > 1 {
		table = table.Where("config_id IN ?", configIDs)
	} else if req.CatalogID != "" {
		// Fallback: treat as filtering expression on config_id
		if c, parseErr := parseAndBuildFilteringQuery(req.CatalogID, "config_id", false); parseErr == nil {
			clauses = append(clauses, c...)
		} else {
			return nil, parseErr
		}
	}

	var output CatalogChangesSearchResponse
	if err := table.Clauses(clauses...).Count(&output.Total).Error; err != nil {
		return nil, err
	}

	if output.Total == 0 {
		timer.Results(output.Changes)
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
	timer.Results(output.Changes)
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
