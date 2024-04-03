package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/timberio/go-datemath"
	"gorm.io/gorm/clause"
)

const (
	CatalogChangeRecursiveUpstream   = "upstream"
	CatalogChangeRecursiveDownstream = "downstream"
	CatalogChangeRecursiveAll        = "all"
)

var allowedConfigChangesSortColumns = []string{"name", "change_type", "summary", "source", "created_at"}

type CatalogChangesSearchRequest struct {
	CatalogID             uuid.UUID `query:"id"`
	ConfigType            string    `query:"config_type"`
	ChangeType            string    `query:"type"`
	Severity              string    `query:"severity"`
	IncludeDeletedConfigs bool      `query:"include_deleted_configs"`
	Depth                 int       `query:"depth"`

	// From date in datemath format
	From string `query:"from"`
	// To date in datemath format
	To string `query:"to"`

	PageSize  int    `query:"page_size"`
	Page      int    `query:"page"`
	SortBy    string `query:"sort_by"`
	sortOrder string

	// upstream | downstream | both
	Recursive string `query:"recursive"`

	fromParsed time.Time
	toParsed   time.Time
}

func (t CatalogChangesSearchRequest) String() string {
	s := ""
	if t.CatalogID != uuid.Nil {
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
	if t.IncludeDeletedConfigs {
		s += fmt.Sprintf("include_deleted_configs: %t ", t.IncludeDeletedConfigs)
	}
	if t.Depth != 0 {
		s += fmt.Sprintf("depth: %d ", t.Depth)
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
	if t.Recursive == "" || t.Recursive == "none" {
		t.Depth = -1
	} else if t.Depth == 0 {
		t.Depth = 5
	}
}

func (t *CatalogChangesSearchRequest) Validate() error {
	if t.Recursive != "" {
		if t.CatalogID == uuid.Nil {
			return fmt.Errorf("recursive must be used with a catalog id")
		}

		if !lo.Contains([]string{CatalogChangeRecursiveUpstream, CatalogChangeRecursiveDownstream, CatalogChangeRecursiveAll}, t.Recursive) {
			return fmt.Errorf("recursive must be one of 'upstream', 'downstream' or 'all'")
		}
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

	return nil
}

type ConfigChangeRow struct {
	ExternalChangeId string     `gorm:"column:external_change_id" json:"external_change_id"`
	ID               string     `gorm:"primaryKey;unique_index;not null;column:id" json:"id"`
	ConfigID         string     `gorm:"column:config_id;default:''" json:"config_id"`
	ChangeType       string     `gorm:"column:change_type" json:"change_type" faker:"oneof:  RunInstances, diff" `
	Severity         string     `gorm:"column:severity" json:"severity"  faker:"oneof: critical, high, medium, low, info"`
	Source           string     `gorm:"column:source" json:"source"`
	Summary          string     `gorm:"column:summary;default:null" json:"summary,omitempty"`
	CreatedAt        *time.Time `gorm:"column:created_at" json:"created_at"`
	ConfigName       string     `gorm:"column:name" json:"name,omitempty"`
	ConfigType       string     `gorm:"column:type" json:"type,omitempty"`
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

func FindCatalogChanges(ctx context.Context, req CatalogChangesSearchRequest) (*CatalogChangesSearchResponse, error) {
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, api.Errorf(api.EINVALID, "bad request: %v", err)
	}
	ctx.Tracef("query changes: %s", req)

	var clauses []clause.Expression

	if req.ConfigType != "" {
		clauses = append(clauses, parseAndBuildFilteringQuery(req.ConfigType, "type")...)
	}

	if req.ChangeType != "" {
		clauses = append(clauses, parseAndBuildFilteringQuery(req.ChangeType, "change_type")...)
	}

	if req.Severity != "" {
		clauses = append(clauses, parseAndBuildFilteringQuery(req.Severity, "severity")...)
	}

	if !req.fromParsed.IsZero() {
		clauses = append(clauses, clause.Gte{Column: clause.Column{Name: "created_at"}, Value: req.fromParsed})
	}

	if !req.toParsed.IsZero() {
		clauses = append(clauses, clause.Lte{Column: clause.Column{Name: "created_at"}, Value: req.toParsed})
	}

	table := ctx.DB().Table("related_changes_recursive(?,?,?,?)", req.CatalogID, req.Recursive, req.IncludeDeletedConfigs, req.Depth)
	if req.CatalogID == uuid.Nil {
		table = ctx.DB().Table("catalog_changes")
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
			clause.OrderBy{Columns: []clause.OrderByColumn{{
				Column: clause.Column{Name: req.SortBy},
				Desc:   req.sortOrder == "desc"},
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
