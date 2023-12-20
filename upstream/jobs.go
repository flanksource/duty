package upstream

import (
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

// SyncCheckStatuses pushes check statuses, that haven't already been pushed, to upstream.
func SyncCheckStatuses(ctx context.Context, config UpstreamConfig, batchSize int) (error, int) {
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
			return fmt.Errorf("failed to fetch check_statuses: %w", err), 0
		}

		if len(checkStatuses) == 0 {
			return nil, count
		}

		logger.Tracef("pushing %d check_statuses to upstream", len(checkStatuses))
		if err := client.Push(ctx, &PushData{AgentName: config.AgentName, CheckStatuses: checkStatuses}); err != nil {
			return fmt.Errorf("failed to push check_statuses to upstream: %w", err), 0
		}

		for i := range checkStatuses {
			checkStatuses[i].IsPushed = true
		}

		if err := ctx.DB().Save(&checkStatuses).Error; err != nil {
			return fmt.Errorf("failed to save check_statuses: %w", err), 0
		}
		count += len(checkStatuses)
	}
}
