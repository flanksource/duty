package upstream

import (
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

// SyncCheckStatuses pushes check statuses, that haven't already been pushed, to upstream.
func SyncCheckStatuses(ctx duty.DBContext, config UpstreamConfig, batchSize int) error {
	var checkStatuses []models.CheckStatus
	if err := ctx.DB().Select("check_statuses.*").
		Joins("Left JOIN checks ON checks.id = check_statuses.check_id").
		Where("checks.agent_id = ?", uuid.Nil).
		Where("check_statuses.is_pushed IS FALSE").
		Find(&checkStatuses).Error; err != nil {
		return fmt.Errorf("failed to fetch checkstatuses: %w", err)
	}

	if len(checkStatuses) == 0 {
		return nil
	}

	logger.Debugf("Pushing %d check statuses to upstream in batches", len(checkStatuses))

	client := NewUpstreamClient(config)

	for i := 0; i < len(checkStatuses); i += batchSize {
		end := i + batchSize
		if end > len(checkStatuses) {
			end = len(checkStatuses)
		}
		batch := checkStatuses[i:end]

		logger.WithValues("batch", fmt.Sprintf("%d/%d", (i/batchSize)+1, (len(checkStatuses)/batchSize)+1)).
			Tracef("Pushing %d check statuses to upstream", len(batch))

		if err := client.Push(ctx, &PushData{AgentName: config.AgentName, CheckStatuses: batch}); err != nil {
			return fmt.Errorf("failed to push check statuses to upstream: %w", err)
		}

		for i := range batch {
			batch[i].IsPushed = true
		}

		if err := ctx.DB().Save(&batch).Error; err != nil {
			return fmt.Errorf("failed to save check statuses: %w", err)
		}
	}

	return nil
}
