package job

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"gorm.io/gorm"
)

func RefreshConfigItemSummary3d(ctx context.Context) error {
	return refreshMatView(ctx, "config_item_summary_3d")
}

func RefreshConfigItemSummary7d(ctx context.Context) error {
	return refreshMatView(ctx, "config_item_summary_7d")
}

func RefreshConfigItemSummary30d(ctx context.Context) error {
	return refreshMatView(ctx, "config_item_summary_30d")
}

func refreshMatView(ctx context.Context, view string) error {
	return ctx.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET ROLE 'postgres'").Error; err != nil {
			return err
		}

		return tx.Exec(fmt.Sprintf("REFRESH MATERIALIZED VIEW %s", view)).Error
	})
}
