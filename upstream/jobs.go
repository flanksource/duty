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

	if c, err := reconcileComponentRelationships(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := reconcileConfigComponentRelationship(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := reconcileCheckComponentRelationship(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := reconcileConfigRelationship(ctx, config, batchSize); err != nil {
		return c, err
	} else {
		count += c
	}

	if c, err := reconcileCheckConfigRelationship(ctx, config, batchSize); err != nil {
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

func reconcileComponentRelationships(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("component_relationships.*").
		Joins("LEFT JOIN components c ON component_relationships.component_id = c.id").
		Joins("LEFT JOIN components rel ON component_relationships.relationship_id = rel.id").
		Where("c.agent_id = ? AND rel.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("component_relationships.is_pushed IS FALSE")

	return reconcileTable[models.ComponentRelationship](ctx, config, fetcher, batchSize)
}

func reconcileConfigComponentRelationship(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("config_component_relationships.*").
		Joins("LEFT JOIN components c ON config_component_relationships.component_id = c.id").
		Joins("LEFT JOIN config_items ci ON config_component_relationships.config_id = ci.id").
		Where("c.agent_id = ? AND ci.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("config_component_relationships.is_pushed IS FALSE")

	return reconcileTable[models.ConfigComponentRelationship](ctx, config, fetcher, batchSize)
}

func reconcileCheckComponentRelationship(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("check_component_relationships.*").
		Joins("LEFT JOIN components c ON check_component_relationships.component_id = c.id").
		Joins("LEFT JOIN canaries ON check_component_relationships.canary_id = canaries.id").
		Where("c.agent_id = ? AND canaries.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("check_component_relationships.is_pushed IS FALSE")

	return reconcileTable[models.CheckComponentRelationship](ctx, config, fetcher, batchSize)
}

func reconcileConfigRelationship(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("config_relationships.*").
		Joins("LEFT JOIN config_items ci ON config_relationships.config_id = ci.id").
		Where("ci.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("config_relationships.is_pushed IS FALSE")

	return reconcileTable[models.ConfigRelationship](ctx, config, fetcher, batchSize)
}

func reconcileCheckConfigRelationship(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	fetcher := ctx.DB().Select("check_config_relationships.*").
		Joins("LEFT JOIN config_items ci ON check_config_relationships.config_id = ci.id").
		Where("ci.agent_id = ?", uuid.Nil, uuid.Nil).
		Where("check_config_relationships.is_pushed IS FALSE")

	return reconcileTable[models.CheckConfigRelationship](ctx, config, fetcher, batchSize)
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

		case "component_relationships":
			ids := lo.Map(items, func(a T, _ int) []string {
				c := any(a).(models.ComponentRelationship)
				return []string{c.ComponentID.String(), c.RelationshipID.String(), c.SelectorID}
			})

			if err := ctx.DB().Model(&models.ComponentRelationship{}).Where("(component_id, relationship_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for component_relationships: %w", err)
			}

		case "config_component_relationships":
			ids := lo.Map(items, func(a T, _ int) []string {
				c := any(a).(models.ConfigComponentRelationship)
				return []string{c.ComponentID.String(), c.ConfigID.String()}
			})

			if err := ctx.DB().Model(&models.ConfigComponentRelationship{}).Where("(component_id, config_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		case "config_relationships":
			ids := lo.Map(items, func(a T, _ int) []string {
				c := any(a).(models.ConfigRelationship)
				return []string{c.RelatedID, c.ConfigID, c.SelectorID}
			})

			if err := ctx.DB().Model(&models.ConfigRelationship{}).Where("(related_id, config_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		case "check_config_relationships":
			ids := lo.Map(items, func(a T, _ int) []string {
				c := any(a).(models.CheckConfigRelationship)
				return []string{c.ConfigID.String(), c.CheckID.String(), c.CanaryID.String(), c.SelectorID}
			})

			if err := ctx.DB().Model(&models.CheckConfigRelationship{}).Where("(config_id, check_id, canary_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		case "check_component_relationships":
			ids := lo.Map(items, func(a T, _ int) []string {
				c := any(a).(models.CheckComponentRelationship)
				return []string{c.ComponentID.String(), c.CheckID.String(), c.CanaryID.String(), c.SelectorID}
			})

			if err := ctx.DB().Model(&models.CheckComponentRelationship{}).Where("(component_id, check_id, canary_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
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
