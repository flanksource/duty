package upstream

import (
	"encoding/json"
	"fmt"

	"github.com/flanksource/duty/api"
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

type parentIsPushedUpdater interface {
	UpdateParentsIsPushed(ctx *gorm.DB, items []models.DBTable) error
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

//
// // TODO: Handle tables with multiple parents
// var reconciledTablesParents = map[string][]pushableTable{
// 	"config_item": {models.ConfigScraper{}},
// 	"check":       {models.Canary{}},
// 	"component":   {models.Topology{}},
//
// 	"config_changes":  {models.ConfigItem{}},
// 	"config_analyses": {models.ConfigItem{}},
// 	"check_status":    {models.Check{}},
//
// 	"check_component_relationships":  {models.Check{}, models.Component{}},
// 	"check_config_relationships":     {models.Check{}},
// 	"component_relationships":        {models.Topology{}},
// 	"config_component_relationships": {models.Topology{}},
// 	"config_relationships":           {models.Topology{}},
// }

func ReconcileAll(ctx context.Context, config UpstreamConfig, batchSize int) (int, error) {
	return ReconcileSome(ctx, config, batchSize)
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
			if a := api.FromError(err); a != nil && a.Data != "" {
				var foreignKeyErr PushFKError
				if err := json.Unmarshal([]byte(a.Data), &foreignKeyErr); err == nil {
					failedOnes := lo.SliceToMap(foreignKeyErr.IDs, func(item string) (string, struct{}) {
						return item, struct{}{}
					})
					failedItems := lo.Filter(items, func(item models.DBTable, _ int) bool {
						_, ok := failedOnes[item.PK()]
						return ok
					})

					if c, ok := table.(parentIsPushedUpdater); ok && len(failedItems) > 0 {
						if err := c.UpdateParentsIsPushed(ctx.DB(), failedItems); err != nil {
							return 0, fmt.Errorf("failed to mark parents as unpushed: %w", err)
						}
					}

					count += len(items) - len(failedItems)
				}
			}

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
