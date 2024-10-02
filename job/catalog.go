package job

import (
	"github.com/flanksource/duty/context"
)

func RefreshConfigItemSummary3d(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW config_item_summary_3d").Error
}

func RefreshConfigItemSummary7d(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW config_item_summary_7d").Error
}

func RefreshConfigItemSummary30d(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW config_item_summary_30d").Error
}
