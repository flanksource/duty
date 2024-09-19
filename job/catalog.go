package job

import (
	"github.com/flanksource/duty/context"
)

func RefreshConfigItemAnalysisChangeCount7d(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW config_item_analysis_change_count_7d").Error
}

func RefreshConfigItemAnalysisChangeCount30d(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW config_item_analysis_change_count_30d").Error
}
