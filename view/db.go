package view

import (
	"encoding/json"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func GetViewColumnDefs(ctx context.Context, namespace, name string) ([]ViewColumnDef, error) {
	var view models.View
	err := ctx.DB().Where("namespace = ? AND name = ?", namespace, name).First(&view).Error
	if err != nil {
		return nil, err
	}

	var spec struct {
		Columns []ViewColumnDef `json:"columns"`
	}

	err = json.Unmarshal(view.Spec, &spec)
	if err != nil {
		return nil, err
	}

	return spec.Columns, nil
}

func GetAllViews(ctx context.Context) ([]models.View, error) {
	var views []models.View
	if err := ctx.DB().Where("deleted_at IS NULL").Find(&views).Error; err != nil {
		return nil, err
	}

	return views, nil
}
