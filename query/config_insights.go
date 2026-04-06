package query

import (
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gorm.io/gorm/clause"
)

type CatalogInsightsSearchRequest struct {
	BaseCatalogSearch `json:",inline"`
	Status            string `query:"status" json:"status"`
	Severity          string `query:"severity" json:"severity"`
	Analyzer          string `query:"analyzer" json:"analyzer"`
	AnalysisType      string `query:"analysis_type" json:"analysis_type"`
}

func (r *CatalogInsightsSearchRequest) SetDefaults() {
	r.BaseCatalogSearch.SetDefaults()
	if r.Status == "" {
		r.Status = "open"
	}
}

type CatalogInsightsSearchResponse struct {
	Total    int64                   `json:"total"`
	Insights []models.ConfigAnalysis `json:"insights"`
}

func FindCatalogInsights(ctx context.Context, req CatalogInsightsSearchRequest) (results *CatalogInsightsSearchResponse, err error) {
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, api.Errorf(api.EINVALID, "bad request: %v", err)
	}

	timer := NewQueryLogger(ctx).Start("CatalogInsights")
	defer timer.End(&err)

	configIDs, err := req.ResolveConfigIDs(ctx)
	if err != nil {
		return nil, err
	}

	var clauses []clause.Expression
	baseClauses, tagsFn := req.ApplyClauses()
	clauses = append(clauses, baseClauses...)

	q := ctx.DB().Table("config_analysis")

	if len(configIDs) > 0 {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: "config_id"}, Value: nil})
		q = q.Where("config_id IN ?", configIDs)
	}

	if req.Status != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.Status, "status", false); parseErr == nil {
			clauses = append(clauses, c...)
		}
	}
	if req.Severity != "" {
		if c, parseErr := parseAndBuildFilteringQuery(formSeverityQuery(req.Severity), "severity", false); parseErr == nil {
			clauses = append(clauses, c...)
		}
	}
	if req.Analyzer != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.Analyzer, "analyzer", true); parseErr == nil {
			clauses = append(clauses, c...)
		}
	}
	if req.AnalysisType != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.AnalysisType, "analysis_type", false); parseErr == nil {
			clauses = append(clauses, c...)
		}
	}

	if tagsFn != nil {
		q = tagsFn(q)
	}

	var output CatalogInsightsSearchResponse
	// Remove the dummy deleted_at clause for config_analysis (it doesn't have deleted_at)
	filteredClauses := make([]clause.Expression, 0, len(clauses))
	for _, c := range clauses {
		if eq, ok := c.(clause.Eq); ok && eq.Column.(clause.Column).Name == "deleted_at" {
			continue
		}
		if eq, ok := c.(clause.Eq); ok && eq.Column.(clause.Column).Name == "config_id" && eq.Value == nil {
			continue
		}
		filteredClauses = append(filteredClauses, c)
	}

	if err := q.Clauses(filteredClauses...).Count(&output.Total).Error; err != nil {
		return nil, err
	}
	if output.Total == 0 {
		timer.Results(output.Insights)
		return &output, nil
	}

	filteredClauses = append(filteredClauses,
		clause.Limit{Limit: &req.PageSize, Offset: (req.Page - 1) * req.PageSize},
	)

	if err := q.Clauses(filteredClauses...).Find(&output.Insights).Error; err != nil {
		return nil, err
	}
	timer.Results(output.Insights)
	return &output, nil
}
