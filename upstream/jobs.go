package upstream

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// SyncCheckStatuses pushes check statuses, that haven't already been pushed, to upstream.
func SyncCheckStatuses(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	client := NewUpstreamClient(config)
	count := 0
	for {
		var checkStatuses []models.CheckStatus
		if err := ctx.DB().Select("check_statuses.*").
			Joins("LEFT JOIN checks ON checks.id = check_statuses.check_id").
			Where("checks.agent_id = ?", uuid.Nil).
			Where("check_statuses.is_pushed IS FALSE").
			Limit(batchSize).
			Find(&checkStatuses).Error; err != nil {
			return 0, fmt.Errorf("failed to fetch check_statuses: %w", err)
		}

		if len(checkStatuses) == 0 {
			return count, nil
		}

		ctx.Tracef("pushing %d check_statuses to upstream", len(checkStatuses))
		if err := client.Push(ctx, &PushData{CheckStatuses: checkStatuses}); err != nil {
			return 0, fmt.Errorf("failed to push check_statuses to upstream: %w", err)
		}

		ids := lo.Map(checkStatuses, func(a models.CheckStatus, _ int) []any { return []any{a.CheckID, a.Time} })
		if err := ctx.DB().Model(&models.CheckStatus{}).Where("(check_id, time) IN ?", ids).Update("is_pushed", true).Error; err != nil {
			return 0, fmt.Errorf("failed to update is_pushed for check_statuses: %w", err)
		}

		count += len(checkStatuses)
	}
}

// ReconcileTable pushes all unpushed items in a table to upstream.
func ReconcileTable[T dbTable](ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	return reconcileTable[T](ctx, config, nil, batchSize)
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

		ids := lo.Map(items, func(a T, _ int) string { return a.PK() })
		if err := ctx.DB().Model(anon).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
			return 0, fmt.Errorf("failed to update is_pushed on %s: %w", table, err)
		}

		count += len(items)
	}
}
