package upstream

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// ReconcilePrecheck, when set, will do an index scan on is_pushed before reconciling
var ReconcilePrecheck = true

func ReconcileAll(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	var count int

	if c, err := ReconcileTable[models.Topology](ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := ReconcileTable[models.ConfigScraper](ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := ReconcileTable[models.Canary](ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := ReconcileTable[models.ConfigItem](ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := ReconcileTable[models.Check](ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := ReconcileTable[models.Component](ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := SyncCheckStatuses(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := SyncConfigAnalyses(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := SyncConfigChanges(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	return count, nil
}

// ReconcileTable pushes all unpushed items in a table to upstream.
func ReconcileTable[T dbTable](ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	return reconcileTable[T](ctx, config, nil, batchSize)
}

// SyncCheckStatuses pushes check statuses, that haven't already been pushed, to upstream.
func SyncCheckStatuses(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("check_statuses.*").
		Joins("LEFT JOIN checks ON checks.id = check_statuses.check_id").
		Where("checks.agent_id = ?", uuid.Nil).
		Where("check_statuses.is_pushed IS FALSE")

	return reconcileTable[models.CheckStatus](ctx, config, fetcher, batchSize)
}

// SyncConfigChanges pushes config changes, that haven't already been pushed, to upstream.
func SyncConfigChanges(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("config_changes.*").
		Joins("LEFT JOIN config_items ON config_items.id = config_changes.config_id").
		Where("config_items.agent_id = ?", uuid.Nil).
		Where("config_changes.is_pushed IS FALSE")

	return reconcileTable[models.ConfigChange](ctx, config, fetcher, batchSize)
}

// SyncConfigAnalyses pushes config analyses, that haven't already been pushed, to upstream.
func SyncConfigAnalyses(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("config_analysis.*").
		Joins("LEFT JOIN config_items ON config_items.id = config_analysis.config_id").
		Where("config_items.agent_id = ?", uuid.Nil).
		Where("config_analysis.is_pushed IS FALSE")

	return reconcileTable[models.ConfigAnalysis](ctx, config, fetcher, batchSize)
}

// ReconcileTable pushes all unpushed items in a table to upstream.
func reconcileTable[T dbTable](ctx context.Context, config UpstreamConfig, fetcher *gorm.DB, batchSize int) (int, error) {
	client := NewUpstreamClient(config)
	var anon T
	table := anon.TableName()

	if ReconcilePrecheck {
		var unpushed float64
		precheck := fmt.Sprintf(`SELECT reltuples FROM pg_class WHERE relname = '%s_is_pushed_idx'`, table)
		if err := ctx.DB().Raw(precheck).Scan(&unpushed).Error; err != nil {
			return 0, fmt.Errorf("failed to check table %q is_pushed index: %w", table, err)
		}

		if unpushed == 0 {
			return 0, nil
		}
	}

	var count int
	for {
		var items []T
		if fetcher != nil {
			if err := fetcher.Limit(batchSize).Find(&items).Error; err != nil {
				return 0, fmt.Errorf("failed to fetch unpushed items for table %s: %w", table, err)
			}
		} else {
			if err := ctx.DB().
				Where("is_pushed IS FALSE").
				Limit(batchSize).
				Find(&items).Error; err != nil {
				return 0, fmt.Errorf("failed to fetch unpushed items for table %s: %w", table, err)
			}
		}

		if len(items) == 0 {
			return count, nil
		}

		ctx.Tracef("pushing %s %d to upstream", table, len(items))
		if err := client.Push(ctx, NewPushData(items)); err != nil {
			return 0, fmt.Errorf("failed to push %s to upstream: %w", table, err)
		}

		switch table {
		case "check_statuses":
			ids := lo.Map(items, func(a T, _ int) []any {
				c := any(a).(models.CheckStatus)
				return []any{c.CheckID, c.Time}
			})

			if err := ctx.DB().Model(&models.CheckStatus{}).Where("(check_id, time) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for check_statuses: %w", err)
			}

		default:
			ids := lo.Map(items, func(a T, _ int) string { return a.PK() })
			if err := ctx.DB().Model(anon).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed on %s: %w", table, err)
			}
		}

		count += len(items)
	}
}
