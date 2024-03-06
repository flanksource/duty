package upstream

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type pushableTable interface {
	models.DBTable
	GetUnpushed(db *gorm.DB) ([]models.DBTable, error)
}

type customIsPushedUpdater interface {
	UpdateIsPushed(db *gorm.DB, items []models.DBTable) error
}

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

		ctx.Tracef("pushing %s %d to upstream", table.TableName(), len(items))
		if err := client.Push(ctx, NewPushData(items)); err != nil {
			return 0, fmt.Errorf("failed to push %s to upstream: %w", table.TableName(), err)
		}
		count += len(items)

		if c, ok := table.(customIsPushedUpdater); ok {
			if err := c.UpdateIsPushed(ctx.DB(), items); err != nil {
				return 0, fmt.Errorf("failed to update is_pushed for %s: %w", table.TableName(), err)
			}
		} else {
			ids := lo.Map(items, func(a models.DBTable, _ int) string { return a.PK() })
			if err := ctx.DB().Model(table).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
				return 0, fmt.Errorf("failed to update is_pushed on %s: %w", table.TableName(), err)
			}
		}
	}
}
