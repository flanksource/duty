package upstream

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
)

var reconciledTables = []pushableTable{
	models.Topology{},
	models.ConfigScraper{},
	models.Canary{},
	models.Artifact{},

	models.ConfigItem{},
	models.Check{},
	models.Component{},

	models.ConfigChange{},
	models.ConfigAnalysis{},
	models.CheckStatus{},

	models.CheckComponentRelationship{},
	models.CheckConfigRelationship{},
	models.ComponentRelationship{},
	models.ConfigComponentRelationship{},
	models.ConfigRelationship{},
}

func ReconcileAll(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	if ctx.Properties().Off("upstream.reconcile.pre-check") {
		return ReconcileSome(ctx, config, batchSize)
	}

	var tablesToReconcile []string
	if err := ctx.DB().Table("unpushed_tables").Scan(&tablesToReconcile).Error; err != nil {
		return 0, err
	}

	return ReconcileSome(ctx, config, batchSize, tablesToReconcile...)
}

func ReconcileSome(ctx context.Context, config UpstreamConfig, batchSize int, runOnly ...string) (int, error) {
	var count int
	for _, table := range reconciledTables {
		if len(runOnly) > 0 && !lo.Contains(runOnly, table.TableName()) {
			ctx.Tracef("skipping reconciliation of table %s", table.TableName())
			continue
		}

		if c, err := reconcileTable(ctx, config, table, batchSize); err != nil {
			return count, fmt.Errorf("failed to reconcile table %s: %w", table.TableName(), err)
		} else {
			count += c
		}
	}

	return count, nil
}

// ReconcileTable pushes all unpushed items in a table to upstream.
func reconcileTable(ctx context.Context, config UpstreamConfig, table pushableTable, batchSize int) (int, error) {
	client := NewUpstreamClient(config)

	var count int
	for {
		items, err := table.GetUnpushed(ctx.DB().Limit(batchSize))
		if err != nil {
			return 0, fmt.Errorf("failed to fetch unpushed items for table %s: %w", table, err)
		}

		if len(items) == 0 {
			return count, nil
		}

		ctx.Tracef("pushing %s %d to upstream", table, len(items))
		if err := client.Push(ctx, NewPushData(items)); err != nil {
			return 0, fmt.Errorf("failed to push %s to upstream: %w", table, err)
		}

		switch table.TableName() {
		case "check_statuses":
			ids := lo.Map(items, func(a models.DBTable, _ int) []any {
				c := any(a).(models.CheckStatus)
				return []any{c.CheckID, c.Time}
			})

			if err := ctx.DB().Model(&models.CheckStatus{}).Where("(check_id, time) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for check_statuses: %w", err)
			}

		case "component_relationships":
			ids := lo.Map(items, func(a models.DBTable, _ int) []string {
				c := any(a).(models.ComponentRelationship)
				return []string{c.ComponentID.String(), c.RelationshipID.String(), c.SelectorID}
			})

			if err := ctx.DB().Model(&models.ComponentRelationship{}).Where("(component_id, relationship_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for component_relationships: %w", err)
			}

		case "config_component_relationships":
			ids := lo.Map(items, func(a models.DBTable, _ int) []string {
				c := any(a).(models.ConfigComponentRelationship)
				return []string{c.ComponentID.String(), c.ConfigID.String()}
			})

			if err := ctx.DB().Model(&models.ConfigComponentRelationship{}).Where("(component_id, config_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		case "config_relationships":
			ids := lo.Map(items, func(a models.DBTable, _ int) []string {
				c := any(a).(models.ConfigRelationship)
				return []string{c.RelatedID, c.ConfigID, c.SelectorID}
			})

			if err := ctx.DB().Model(&models.ConfigRelationship{}).Where("(related_id, config_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		case "check_config_relationships":
			ids := lo.Map(items, func(a models.DBTable, _ int) []string {
				c := any(a).(models.CheckConfigRelationship)
				return []string{c.ConfigID.String(), c.CheckID.String(), c.CanaryID.String(), c.SelectorID}
			})

			if err := ctx.DB().Model(&models.CheckConfigRelationship{}).Where("(config_id, check_id, canary_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		case "check_component_relationships":
			ids := lo.Map(items, func(a models.DBTable, _ int) []string {
				c := any(a).(models.CheckComponentRelationship)
				return []string{c.ComponentID.String(), c.CheckID.String(), c.CanaryID.String(), c.SelectorID}
			})

			if err := ctx.DB().Model(&models.CheckComponentRelationship{}).Where("(component_id, check_id, canary_id, selector_id) IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for config_component_relationships: %w", err)
			}

		default:
			ids := lo.Map(items, func(a models.DBTable, _ int) string { return a.PK() })
			if err := ctx.DB().Model(table).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed on %s: %w", table, err)
			}
		}

		count += len(items)
	}
}
