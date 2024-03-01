package upstream

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
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

		for i := range checkStatuses {
			checkStatuses[i].IsPushed = true
		}

		if err := ctx.DB().Save(&checkStatuses).Error; err != nil {
			return 0, fmt.Errorf("failed to save check_statuses: %w", err)
		}
		count += len(checkStatuses)
	}
}

// SyncConfigChanges pushes config changes, that haven't already been pushed, to upstream.
func SyncConfigChanges(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	client := NewUpstreamClient(config)
	count := 0
	for {
		var configChanges []models.ConfigChange
		if err := ctx.DB().Select("config_changes.*").
			Joins("LEFT JOIN config_items ON config_items.id = config_changes.config_id").
			Where("config_items.agent_id = ?", uuid.Nil).
			Where("config_changes.is_pushed IS FALSE").
			Limit(batchSize).
			Find(&configChanges).Error; err != nil {
			return 0, fmt.Errorf("failed to fetch config_changes: %w", err)
		}

		if len(configChanges) == 0 {
			return count, nil
		}

		ctx.Tracef("pushing %d config_changes to upstream", len(configChanges))
		if err := client.Push(ctx, &PushData{ConfigChanges: configChanges}); err != nil {
			return 0, fmt.Errorf("failed to push config_changes to upstream: %w", err)
		}

		ids := lo.Map(configChanges, func(c models.ConfigChange, _ int) string { return c.ID })
		if err := ctx.DB().Model(&models.ConfigChange{}).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
			return 0, fmt.Errorf("failed to update is_pushed on config_changes: %w", err)
		}

		count += len(configChanges)
	}
}

// SyncConfigAnalyses pushes config analyses, that haven't already been pushed, to upstream.
func SyncConfigAnalyses(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	client := NewUpstreamClient(config)
	count := 0
	for {
		var analyses []models.ConfigAnalysis
		if err := ctx.DB().Select("config_analysis.*").
			Joins("LEFT JOIN config_items ON config_items.id = config_analysis.config_id").
			Where("config_items.agent_id = ?", uuid.Nil).
			Where("config_analysis.is_pushed IS FALSE").
			Limit(batchSize).
			Find(&analyses).Error; err != nil {
			return 0, fmt.Errorf("failed to fetch config_analysis: %w", err)
		}

		if len(analyses) == 0 {
			return count, nil
		}

		ctx.Tracef("pushing %d config_analyses to upstream", len(analyses))
		if err := client.Push(ctx, &PushData{ConfigAnalysis: analyses}); err != nil {
			return 0, fmt.Errorf("failed to push config_analysis to upstream: %w", err)
		}

		ids := lo.Map(analyses, func(a models.ConfigAnalysis, _ int) string { return a.ID.String() })
		if err := ctx.DB().Model(&models.ConfigAnalysis{}).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
			return 0, fmt.Errorf("failed to update is_pushed on config_analysis: %w", err)
		}

		count += len(analyses)
	}
}

// SyncArtifacts pushes artifacts that haven't already been pushed to upstream.
func SyncArtifacts(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	client := NewUpstreamClient(config)
	count := 0
	for {
		var artifacts []models.Artifact
		if err := ctx.DB().
			Where("is_pushed IS FALSE").
			Limit(batchSize).
			Find(&artifacts).Error; err != nil {
			return 0, fmt.Errorf("failed to fetch artifacts: %w", err)
		}

		if len(artifacts) == 0 {
			return count, nil
		}

		ctx.Tracef("pushing %d artifacts to upstream", len(artifacts))
		if err := client.Push(ctx, &PushData{Artifacts: artifacts}); err != nil {
			return 0, fmt.Errorf("failed to push artifacts to upstream: %w", err)
		}

		ids := lo.Map(artifacts, func(a models.Artifact, _ int) string { return a.ID.String() })
		if err := ctx.DB().Model(&models.Artifact{}).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
			return 0, fmt.Errorf("failed to update is_pushed on artifacts: %w", err)
		}

		count += len(artifacts)
	}
}
