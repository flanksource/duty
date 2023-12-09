package upstream

import (
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

// SyncCheckStatuses pushes check statuses, that haven't already been pushed, to upstream.
func SyncCheckStatuses(ctx context.Context, config UpstreamConfig, batchSize int) error {
	client := NewUpstreamClient(config)

	for {
		var checkStatuses []models.CheckStatus
		if err := ctx.DB().Select("check_statuses.*").
			Joins("LEFT JOIN checks ON checks.id = check_statuses.check_id").
			Where("checks.agent_id = ?", uuid.Nil).
			Where("check_statuses.is_pushed IS FALSE").
			Limit(batchSize).
			Find(&checkStatuses).Error; err != nil {
			return fmt.Errorf("failed to fetch checkstatuses: %w", err)
		}

		if len(checkStatuses) == 0 {
			logger.Debugf("no check statuses to push")
			return nil
		}

		logger.Tracef("pushing %d check statuses to upstream", len(checkStatuses))
		if err := client.Push(ctx, &PushData{AgentName: config.AgentName, CheckStatuses: checkStatuses}); err != nil {
			return fmt.Errorf("failed to push check statuses to upstream: %w", err)
		}

		for i := range checkStatuses {
			checkStatuses[i].IsPushed = true
		}

		if err := ctx.DB().Save(&checkStatuses).Error; err != nil {
			return fmt.Errorf("failed to save check statuses: %w", err)
		}
	}
}
