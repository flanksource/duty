package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/timberio/go-datemath"
)

const (
	CatalogChangeRecursiveUpstream   = "upstream"
	CatalogChangeRecursiveDownstream = "downstream"
	CatalogChangeRecursiveBoth       = "both"
)

type CatalogChangesSearchRequest struct {
	CatalogID             uuid.UUID `query:"id"`
	ConfigType            string    `query:"config_type"`
	ChangeType            string    `query:"type"`
	Severity              string    `query:"severity"`
	IncludeDeletedConfigs bool      `query:"include_deleted_configs"`

	// From date in datemath format
	From string `query:"from"`
	// To date in datemath format
	To string `query:"to"`

	PageSize int    `query:"page_size"`
	Page     int    `query:"page"`
	SortBy   string `query:"sort_by"`

	// upstream | downstream | both
	Recursive string `query:"recursive"`

	fromParsed time.Time
	toParsed   time.Time
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
}

func (t *CatalogChangesSearchRequest) Validate() error {
	if t.CatalogID == uuid.Nil {
		return fmt.Errorf("catalog id is required")
	}

	if t.Recursive != "" && !lo.Contains([]string{CatalogChangeRecursiveUpstream, CatalogChangeRecursiveDownstream, CatalogChangeRecursiveBoth}, t.Recursive) {
		return fmt.Errorf("recursive must be one of 'upstream', 'downstream' or 'both'")
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

	return nil
}

type CatalogChangesSearchResponse struct {
	Summary map[string]int        `json:"summary,omitempty"`
	Changes []models.ConfigChange `json:"changes,omitempty"`
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

	args := map[string]any{
		"catalog_id":              req.CatalogID,
		"recursive":               req.Recursive,
		"include_deleted_configs": req.IncludeDeletedConfigs,
	}

	var clauses []string
	query := "SELECT cc.* FROM related_changes_recursive(@catalog_id, @recursive, @include_deleted_configs) cc"
	if req.Recursive == "" {
		query = "SELECT cc.* FROM config_changes cc"
		clauses = append(clauses, "cc.config_id = @catalog_id")
	}

	if req.ConfigType != "" {
		query += " LEFT JOIN config_items ON cc.config_id = config_items.id"

		_clauses, _args := parseAndBuildFilteringQuery(req.ConfigType, "config_items.type")
		clauses = append(clauses, _clauses...)
		args = collections.MergeMap(args, _args)
	}

	if req.ChangeType != "" {
		_clauses, _args := parseAndBuildFilteringQuery(req.ChangeType, "cc.change_type")
		clauses = append(clauses, _clauses...)
		args = collections.MergeMap(args, _args)
	}

	if req.Severity != "" {
		_clauses, _args := parseAndBuildFilteringQuery(req.Severity, "cc.severity")
		clauses = append(clauses, _clauses...)
		args = collections.MergeMap(args, _args)
	}

	if !req.fromParsed.IsZero() {
		clauses = append(clauses, "cc.created_at > @from")
		args["from"] = req.fromParsed
	}

	if !req.toParsed.IsZero() {
		clauses = append(clauses, "cc.created_at < @to")
		args["to"] = req.toParsed
	}

	if len(clauses) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(clauses, " AND "))
	}

	if req.SortBy != "" {
		var sortOrder = "ASC"
		if strings.HasPrefix(string(req.SortBy), "-") {
			sortOrder = "DESC"
			req.SortBy = strings.TrimPrefix(string(req.SortBy), "-")
		}

		query += fmt.Sprintf(" ORDER BY @sortby %s", sortOrder)
		args["sortby"] = req.SortBy
	}

	query += " LIMIT @page_size OFFSET @page_number"
	args["page_size"] = req.PageSize
	args["page_number"] = (req.Page - 1) * req.PageSize

	var output CatalogChangesSearchResponse
	if err := ctx.DB().Raw(query, args).Find(&output.Changes).Error; err != nil {
		return nil, err
	}

	output.Summarize()
	return &output, nil
}