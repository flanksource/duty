package upstream

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	dutil "github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
	"github.com/sethvargo/go-retry"
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

// Compile time check to ensure that tables with parent implement this interface.
var (
	_ parentIsPushedUpdater = (*models.ConfigItem)(nil)
	_ parentIsPushedUpdater = (*models.ConfigChange)(nil)
	_ parentIsPushedUpdater = (*models.ConfigChange)(nil)
	_ parentIsPushedUpdater = (*models.ConfigAnalysis)(nil)
	_ parentIsPushedUpdater = (*models.ConfigRelationship)(nil)

	_ parentIsPushedUpdater = (*models.Component)(nil)
	_ parentIsPushedUpdater = (*models.ComponentRelationship)(nil)
	_ parentIsPushedUpdater = (*models.ConfigComponentRelationship)(nil)

	_ parentIsPushedUpdater = (*models.Check)(nil)
	_ parentIsPushedUpdater = (*models.CheckStatus)(nil)
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

	models.JobHistory{},
}

func ReconcileAll(ctx context.Context, config UpstreamConfig, batchSize int) (int, int, error) {
	return ReconcileSome(ctx, config, batchSize)
}

func ReconcileSome(ctx context.Context, config UpstreamConfig, batchSize int, runOnly ...string) (int, int, error) {
	var count, fkFailed int
	for _, table := range reconciledTables {
		if len(runOnly) > 0 && !lo.Contains(runOnly, table.TableName()) {
			continue
		}

		success, failed, err := reconcileTable(ctx, config, table, batchSize)
		count += success
		fkFailed += failed
		if err != nil {
			return count, fkFailed, fmt.Errorf("failed to reconcile table %s: %w", table.TableName(), err)
		}
	}

	return count, fkFailed, nil
}

// ReconcileTable pushes all unpushed items in a table to upstream.
func reconcileTable(ctx context.Context, config UpstreamConfig, table pushableTable, batchSize int) (int, int, error) {
	client := NewUpstreamClient(config)

	var count, fkFailed int
	for {
		items, err := table.GetUnpushed(ctx.DB().Limit(batchSize))
		if err != nil {
			return count, fkFailed, fmt.Errorf("failed to fetch unpushed items for table %s: %w", table, err)
		}

		if len(items) == 0 {
			return count, fkFailed, nil
		}

		ctx.Tracef("pushing %s %d to upstream", table.TableName(), len(items))
		pushError := client.Push(ctx, NewPushData(items))
		if pushError != nil {
			httpError := api.HTTPErrorFromErr(pushError)
			if httpError == nil || httpError.Data == "" {
				return count, fkFailed, fmt.Errorf("failed to push %s to upstream: %w", table.TableName(), pushError)
			}

			var foreignKeyErr PushFKError
			if err := json.Unmarshal([]byte(httpError.Data), &foreignKeyErr); err != nil {
				return count, fkFailed, fmt.Errorf("failed to push %s to upstream (could not decode api error: %w): %w", table.TableName(), err, pushError)
			}

			failedOnes := lo.SliceToMap(foreignKeyErr.IDs, func(item string) (string, struct{}) {
				return item, struct{}{}
			})
			failedItems := lo.Filter(items, func(item models.DBTable, _ int) bool {
				_, ok := failedOnes[item.PK()]
				return ok
			})
			fkFailed += len(failedItems)

			if c, ok := table.(parentIsPushedUpdater); ok && len(failedItems) > 0 {
				if err := c.UpdateParentsIsPushed(ctx.DB(), failedItems); err != nil {
					return count, fkFailed, fmt.Errorf("failed to mark parents as unpushed: %w", err)
				}
			}

			items = lo.Filter(items, func(item models.DBTable, _ int) bool {
				_, ok := failedOnes[item.PK()]
				return !ok
			})
		}

		count += len(items)

		batchSize := ctx.Properties().Int("update_is_pushed.batch.size", 200)
		for _, batch := range lo.Chunk(items, batchSize) {
			backoff := retry.WithJitter(time.Second, retry.WithMaxRetries(3, retry.NewExponential(time.Second)))
			err = retry.Do(ctx, backoff, func(_ctx gocontext.Context) error {
				ctx = _ctx.(context.Context)

				if c, ok := table.(customIsPushedUpdater); ok {
					if err := c.UpdateIsPushed(ctx.DB(), batch); err != nil {
						if dutil.IsDeadlockError(err) {
							return retry.RetryableError(err)
						}

						return fmt.Errorf("failed to update is_pushed on %s: %w", table.TableName(), err)
					}
				} else {
					ids := lo.Map(batch, func(a models.DBTable, _ int) string { return a.PK() })
					if err := ctx.DB().Model(table).Where("id IN ?", ids).Update("is_pushed", true).Error; err != nil {
						if dutil.IsDeadlockError(err) {
							return retry.RetryableError(err)
						}

						return fmt.Errorf("failed to update is_pushed on %s: %w", table.TableName(), err)
					}
				}

				return nil
			})
			if err != nil {
				return count, fkFailed, err
			}
		}

		if pushError != nil {
			return count, fkFailed, pushError
		}
	}
}
