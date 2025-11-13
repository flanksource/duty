package upstream

import (
	gocontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/properties"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"github.com/sethvargo/go-retry"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	dutil "github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
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

// Tables whose primary key is not just the "id" column need to implement this interface.
var (
	_ customIsPushedUpdater = (*models.CheckStatus)(nil)
	_ customIsPushedUpdater = (*models.ConfigRelationship)(nil)
	_ customIsPushedUpdater = (*models.ConfigComponentRelationship)(nil)
	_ customIsPushedUpdater = (*models.CheckComponentRelationship)(nil)
	_ customIsPushedUpdater = (*models.CheckConfigRelationship)(nil)
	_ customIsPushedUpdater = (*models.GeneratedViewTable)(nil)
	_ customIsPushedUpdater = (*models.ConfigItemLastScrapedTime)(nil)
)

type ForeignKeyErrorSummary struct {
	ids []string
}

func (fks ForeignKeyErrorSummary) Count() int     { return len(fks.ids) }
func (fks *ForeignKeyErrorSummary) Add(id string) { fks.ids = append(fks.ids, id) }

const FKErrorIDCount = 10

func (fks ForeignKeyErrorSummary) MarshalJSON() ([]byte, error) {
	count := len(fks.ids)
	if count == 0 {
		return []byte(`null`), nil
	}
	// Display less IDs to keep UI consistent
	idLimit := properties.Int(FKErrorIDCount, "upstream.summary.fkerror_id_count")
	fks.ids = lo.Slice(fks.ids, 0, idLimit)
	if len(fks.ids) >= idLimit {
		fks.ids = append(fks.ids, "...")
	}
	return json.Marshal(map[string]any{"ids": fks.ids, "count": count})
}

type ReconcileTableSummary struct {
	Success   int                    `json:"success,omitempty"`
	FKeyError ForeignKeyErrorSummary `json:"foreign_error,omitempty"`
	Skipped   bool                   `json:"skipped,omitempty"`
	Error     *oops.OopsError        `json:"error,omitempty"`
}

type ReconcileSummary map[string]ReconcileTableSummary

// DidReconcile returns true if all of the given tables
// reconciled successfully.
func (t ReconcileSummary) DidReconcile(tables []string) bool {
	if len(tables) == 0 {
		return true
	}

	if t == nil {
		return false // nothing has been reconciled yet
	}

	for _, table := range tables {
		summary, ok := t[table]
		if !ok {
			return false // this table hasn't been reconciled yet
		}

		reconciled := !summary.Skipped && summary.Error == nil && summary.FKeyError.Count() == 0
		if !reconciled {
			return false // table didn't reconcile successfully
		}
	}

	return true
}

func (t ReconcileSummary) GetSuccessFailure() (int, int) {
	var success, failure int
	for _, summary := range t {
		success += summary.Success
		failure += summary.FKeyError.Count()
	}
	return success, failure
}

func (t *ReconcileSummary) AddSkipped(tables ...pushableTable) {
	if t == nil || (*t) == nil {
		(*t) = make(ReconcileSummary)
	}

	for _, table := range tables {
		v := (*t)[table.TableName()]
		v.Skipped = true
		(*t)[table.TableName()] = v
	}
}

func (t *ReconcileSummary) AddStat(table string, success int, foreignKeyFailures ForeignKeyErrorSummary, err error) {
	if success == 0 && foreignKeyFailures.Count() == 0 && err == nil {
		return
	}

	if t == nil || (*t) == nil {
		(*t) = make(ReconcileSummary)
	}

	v := (*t)[table]
	v.Success = success
	v.FKeyError = foreignKeyFailures
	if err != nil {
		// For json marshaling
		v.Error = lo.ToPtr(oops.Wrap(err).(oops.OopsError))
	}

	(*t)[table] = v
}

func (t ReconcileSummary) Error() error {
	var allErrors []string
	for table, summary := range t {
		if summary.Error != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: %s; ", table, summary.Error))
		}

		if summary.FKeyError.Count() > 0 {
			allErrors = append(allErrors, fmt.Sprintf("%s: %d foreign key errors; ", table, summary.Error))
		}
	}

	if len(allErrors) == 0 {
		return nil
	}

	return errors.New(strings.Join(allErrors, ";"))
}

// PushGroup are a set of tables that need to be reconciled in order.
// If one fails, the rest are skipped.
type PushGroup struct {
	Name   string
	Tables []pushableTable

	// DependsOn is a list of tables that need to be reconciled
	// for this group to be reconciled.
	DependsOn []string
}

const generatedViewsGroup = "generated_views"

var reconcileTableGroups = []PushGroup{
	{
		Name: "configs",
		Tables: []pushableTable{
			models.ConfigScraper{}, models.ConfigItem{}, models.ConfigItemLastScrapedTime{},
			models.ConfigChange{}, models.ConfigAnalysis{}, models.ConfigRelationship{}},
	},
	{
		Name:   "topologies",
		Tables: []pushableTable{models.Topology{}, models.Component{}, models.ComponentRelationship{}},
	},
	{
		Name:   "canaries",
		Tables: []pushableTable{models.Canary{}, models.Check{}, models.CheckStatus{}},
	},
	{
		Name:      "CheckComponentRelationship",
		Tables:    []pushableTable{models.CheckComponentRelationship{}},
		DependsOn: []string{models.Check{}.TableName(), models.Component{}.TableName()},
	},
	{
		Name:      "CheckConfigRelationship",
		Tables:    []pushableTable{models.CheckConfigRelationship{}},
		DependsOn: []string{models.Check{}.TableName(), models.ConfigItem{}.TableName()},
	},
	{
		Name:      "ConfigComponentRelationship",
		Tables:    []pushableTable{models.ConfigComponentRelationship{}},
		DependsOn: []string{models.ConfigItem{}.TableName(), models.Component{}.TableName()},
	},
	{
		Name:   "JobHistory",
		Tables: []pushableTable{models.JobHistory{}},
	},
	{
		Name:   "Artifact",
		Tables: []pushableTable{models.Artifact{}},
	},
	{
		Name:   "ViewPanels",
		Tables: []pushableTable{models.ViewPanel{}},
	},
}

func ReconcileAll(ctx context.Context, client *UpstreamClient, batchSize int) ReconcileSummary {
	return ReconcileSome(ctx, client, batchSize)
}

func ReconcileSome(ctx context.Context, client *UpstreamClient, batchSize int, runOnly ...string) ReconcileSummary {
	var summary ReconcileSummary

	reconcileTableGroupsCopy, err := reconcileTableGroupsWithGeneratedViews(ctx, client)
	if err != nil {
		summary.AddStat("generated_view_tables", 0, ForeignKeyErrorSummary{}, err)
		return summary
	}

	for _, group := range reconcileTableGroupsCopy {
		if !summary.DidReconcile(group.DependsOn) {
			summary.AddSkipped(group.Tables...)
			continue
		}

	outer:
		for i, table := range group.Tables {
			if len(runOnly) > 0 && !lo.Contains(runOnly, table.TableName()) {
				continue
			}

			success, failed, err := reconcileTable(ctx, client, table, batchSize)
			summary.AddStat(table.TableName(), success, failed, err)
			if err != nil {
				if i != len(group.Tables)-1 {
					// If there are remaining tables in this group, skip them.
					summary.AddSkipped(group.Tables[i+1:]...)
				}

				break outer
			}
		}
	}

	return summary
}

// ReconcileTable pushes all unpushed items in a table to upstream.
func reconcileTable(ctx context.Context, client *UpstreamClient, table pushableTable, batchSize int) (int, ForeignKeyErrorSummary, error) {
	var count int
	var fkFailed ForeignKeyErrorSummary
	for {
		items, err := table.GetUnpushed(ctx.DB().Limit(batchSize))
		if err != nil {
			return count, fkFailed, fmt.Errorf("failed to fetch unpushed items for table %s: %w", table, err)
		}

		if len(items) == 0 {
			return count, fkFailed, nil
		}

		var fkErrorOccured bool

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

			fkErrorOccured = !foreignKeyErr.Empty()

			failedOnes := lo.SliceToMap(foreignKeyErr.IDs, func(item string) (string, struct{}) {
				return item, struct{}{}
			})
			failedItems := lo.Filter(items, func(item models.DBTable, _ int) bool {
				_, ok := failedOnes[item.PK()]
				if ok {
					fkFailed.Add(item.PK())
				}
				return ok
			})

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

		if fkErrorOccured {
			// we stop reconciling for this table.
			// In the next reconciliation run, the parents will be pushed
			// and the fk error will resolve then.
			return count, fkFailed, nil
		}

		if pushError != nil {
			return count, fkFailed, pushError
		}
	}
}

func ResetIsPushed(ctx context.Context) error {
	intervalDays := ctx.Properties().Int("job.ResetIsPushed.interval_days", 7)

	overrides := map[string]string{
		models.ConfigAnalysis{}.TableName(): fmt.Sprintf(`first_observed >= NOW() - INTERVAL '%d days' OR last_observed >= NOW() - INTERVAL '%d days'`, intervalDays, intervalDays),
		models.ConfigChange{}.TableName():   fmt.Sprintf(`created_at >= NOW() - INTERVAL '%d days'`, intervalDays),
		models.CheckStatus{}.TableName():    fmt.Sprintf(`created_at >= NOW() - INTERVAL '%d days'`, intervalDays),
		models.JobHistory{}.TableName():     fmt.Sprintf(`time_start >= NOW() - INTERVAL '%d days'`, intervalDays),
		models.ViewPanel{}.TableName():      fmt.Sprintf(`refreshed_at >= NOW() - INTERVAL '%d days'`, intervalDays),
	}

	defQuery := fmt.Sprintf(`created_at >= NOW() - INTERVAL '%d days' OR updated_at >= NOW() - INTERVAL '%d days'`, intervalDays, intervalDays)

	if !ctx.Properties().On(false, "job.ResetIsPushed.ignore_deleted_at") {
		defQuery += " AND deleted_at IS NULL"
	}

	var errs []error
	for _, pg := range reconcileTableGroups {
		for _, table := range pg.Tables {
			// None of the override tables have deleted_at field so it can be ignored
			if err := ctx.DB().Table(table.TableName()).
				Where(lo.CoalesceOrEmpty(overrides[table.TableName()], defQuery)).
				Update("is_pushed", false).Error; err != nil {
				errs = append(errs, fmt.Errorf("error updating is_pushed for table[%s]: %w", table.TableName(), err))
			}
		}
	}

	return oops.Join(errs...)
}
