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
	if len(configIDs) == 0 && req.CatalogID != "" {
		return &CatalogInsightsSearchResponse{}, nil
	}

	// config_analysis table doesn't have deleted_at, agent_id, type, or tags columns
	baseClauses, tagsFn, err := req.ApplyClauses("deleted_at", "agent_id", "type", "tags")
	if err != nil {
		return nil, api.Errorf(api.EINVALID, "bad request: %v", err)
	}
	var clauses []clause.Expression
	clauses = append(clauses, baseClauses...)

	q := ctx.DB().Table("config_analysis")

	if len(configIDs) > 0 {
		q = q.Where("config_id IN ?", configIDs)
	}

	if req.Status != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.Status, "status", false); parseErr != nil && !req.Lenient {
			return nil, api.Errorf(api.EINVALID, "failed to parse status: %v", parseErr)
		} else if parseErr == nil {
			clauses = append(clauses, c...)
		}
	}
	if req.Severity != "" {
		severityQuery, err := formSeverityQuery(req.Severity)
		if err != nil && !req.Lenient {
			return nil, api.Errorf(api.EINVALID, "invalid severity: %v", err)
		} else if err == nil {
			if c, parseErr := parseAndBuildFilteringQuery(severityQuery, "severity", false); parseErr != nil && !req.Lenient {
				return nil, api.Errorf(api.EINVALID, "failed to parse severity: %v", parseErr)
			} else if parseErr == nil {
				clauses = append(clauses, c...)
			}
		}
	}
	if req.Analyzer != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.Analyzer, "analyzer", true); parseErr != nil && !req.Lenient {
			return nil, api.Errorf(api.EINVALID, "failed to parse analyzer: %v", parseErr)
		} else if parseErr == nil {
			clauses = append(clauses, c...)
		}
	}
	if req.AnalysisType != "" {
		if c, parseErr := parseAndBuildFilteringQuery(req.AnalysisType, "analysis_type", false); parseErr != nil && !req.Lenient {
			return nil, api.Errorf(api.EINVALID, "failed to parse analysis_type: %v", parseErr)
		} else if parseErr == nil {
			clauses = append(clauses, c...)
		}
	}

	if tagsFn != nil {
		q = tagsFn(q)
	}

	var output CatalogInsightsSearchResponse
	if err := q.Clauses(clauses...).Count(&output.Total).Error; err != nil {
		return nil, err
	}
	if output.Total == 0 {
		timer.Results(output.Insights)
		return &output, nil
	}

	clauses = append(clauses,
		clause.Limit{Limit: &req.PageSize, Offset: (req.Page - 1) * req.PageSize},
	)

	if err := q.Clauses(clauses...).Find(&output.Insights).Error; err != nil {
		return nil, err
	}
	timer.Results(output.Insights)
	return &output, nil
}
