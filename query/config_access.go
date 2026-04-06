package query

import (
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

type CatalogAccessSearchRequest struct {
	BaseCatalogSearch `json:",inline"`
}

type CatalogAccessSearchResponse struct {
	Total  int64                          `json:"total"`
	Access []models.ConfigAccessSummary   `json:"access"`
}

func FindCatalogAccess(ctx context.Context, req CatalogAccessSearchRequest) (results *CatalogAccessSearchResponse, err error) {
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, api.Errorf(api.EINVALID, "bad request: %v", err)
	}

	timer := NewQueryLogger(ctx).Start("CatalogAccess")
	defer timer.End(&err)

	configIDs, err := req.ResolveConfigIDs(ctx)
	if err != nil {
		return nil, err
	}

	var output CatalogAccessSearchResponse
	q := ctx.DB().Table("config_access_summary")
	if len(configIDs) > 0 {
		q = q.Where("config_id IN ?", configIDs)
	}

	if err := q.Count(&output.Total).Error; err != nil {
		return nil, err
	}
	if output.Total == 0 {
		timer.Results(output.Access)
		return &output, nil
	}

	if err := q.Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(&output.Access).Error; err != nil {
		return nil, err
	}
	timer.Results(output.Access)
	return &output, nil
}

func FindConfigAccessByConfigIDs(ctx context.Context, configIDs []uuid.UUID) ([]models.ConfigAccessSummary, error) {
	resp, err := FindCatalogAccess(ctx, CatalogAccessSearchRequest{
		BaseCatalogSearch: BaseCatalogSearch{configIDs: configIDs},
	})
	if err != nil {
		return nil, err
	}
	return resp.Access, nil
}
