package upstream

import (
	"errors"
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	dutil "github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
)

const (
	saveRetries = 3
)

type PushFKError struct {
	IDs []string `json:"ids"`
}

func (t *PushFKError) Empty() bool {
	return len(t.IDs) == 0
}

func getAgent(ctx context.Context, name string) (*models.Agent, error) {
	var t models.Agent
	tx := ctx.DB().Where("name = ?", name).Where("deleted_at IS NULL").First(&t)
	return &t, tx.Error
}

func createAgent(ctx context.Context, name string) (*models.Agent, error) {
	a := models.Agent{Name: name}
	tx := ctx.DB().Create(&a)
	return &a, tx.Error
}

func GetOrCreateAgent(ctx context.Context, name string) (*models.Agent, error) {
	a, err := getAgent(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newAgent, err := createAgent(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent: %w", err)
			}
			return newAgent, nil
		}
		return nil, err
	}

	return a, nil
}

// DeleteOnUpstream deletes the given resources by agent on the upstream.
func DeleteOnUpstream(ctx context.Context, req *PushData) error {
	db := ctx.DB()

	if len(req.ComponentRelationships) > 0 {
		if err := db.Delete(req.ComponentRelationships).Error; err != nil {
			return fmt.Errorf("error deleting component_relationships: %w", err)
		}
	}

	if len(req.ConfigRelationships) > 0 {
		if err := db.Delete(req.ConfigRelationships).Error; err != nil {
			return fmt.Errorf("error deleting config_relationships: %w", err)
		}
	}

	if len(req.ConfigComponentRelationships) > 0 {
		if err := db.Delete(req.ConfigComponentRelationships).Error; err != nil {
			return fmt.Errorf("error deleting config_component_relationships: %w: %+v", err, req.ConfigComponentRelationships)
		}
	}

	if len(req.ConfigChanges) > 0 {
		if err := db.Delete(req.ConfigChanges).Error; err != nil {
			return fmt.Errorf("error deleting config changes: %w", err)
		}
	}

	if len(req.ConfigAnalysis) > 0 {
		if err := db.Delete(req.ConfigAnalysis).Error; err != nil {
			return fmt.Errorf("error deleting config analysis: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		if err := db.Delete(req.CheckStatuses).Error; err != nil {
			return fmt.Errorf("error deleting check_statuses: %w", err)
		}
	}

	if len(req.Components) > 0 {
		if err := db.Delete(req.Components).Error; err != nil {
			logger.Errorf("error deleting components: %w", err)
		}
	}

	if len(req.ConfigItems) > 0 {
		if err := db.Delete(req.ConfigItems).Error; err != nil {
			logger.Errorf("error deleting config items: %w", err)
		}
	}

	if len(req.Checks) > 0 {
		if err := db.Delete(req.Checks).Error; err != nil {
			return fmt.Errorf("error deleting checks: %w", err)
		}
	}

	if len(req.ConfigScrapers) > 0 {
		if err := db.Delete(req.ConfigScrapers).Error; err != nil {
			return fmt.Errorf("error deleting config scrapers: %w", err)
		}
	}

	if len(req.Canaries) > 0 {
		if err := db.Delete(req.Canaries).Error; err != nil {
			return fmt.Errorf("error deleting canaries: %w", err)
		}
	}

	if len(req.Topologies) > 0 {
		if err := db.Delete(req.Topologies).Error; err != nil {
			return fmt.Errorf("error deleting topologies: %w", err)
		}
	}

	if len(req.JobHistory) > 0 {
		if err := db.Delete(req.JobHistory).Error; err != nil {
			return fmt.Errorf("error deleting job_history: %w", err)
		}
	}

	if len(req.ViewPanels) > 0 {
		if err := db.Delete(req.ViewPanels).Error; err != nil {
			return fmt.Errorf("error deleting view_panels: %w", err)
		}
	}

	if len(req.GeneratedViews) > 0 {
		for _, viewData := range req.GeneratedViews {
			if err := deleteViewData(ctx, viewData); err != nil {
				return fmt.Errorf("error deleting view_data: %w", err)
			}
		}
	}

	return nil
}

func InsertUpstreamMsg(ctx context.Context, req *PushData) error {
	batchSize := 100
	db := ctx.DB()
	for _, c := range req.Topologies {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Omit("created_by").Create(&c).Error; err != nil {
			return fmt.Errorf("error upserting topology: (id=%s): %w", c.ID, err)
		}
	}

	for _, c := range req.Canaries {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Omit("created_by").Create(&c).Error; err != nil {
			return fmt.Errorf("error upserting canaries: (id=%s): %w", c.ID, err)
		}
	}

	// components are inserted one by one, instead of in a batch, because of the foreign key constraint with itself.
	if err := saveIndividuallyWithRetries(ctx, req.Components, saveRetries); err != nil {
		return fmt.Errorf("error upserting components: %w", err)
	}

	if len(req.ComponentRelationships) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: models.ComponentRelationship{}.PKCols()}).CreateInBatches(req.ComponentRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.ComponentRelationships, func(i models.ComponentRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	for _, c := range req.ConfigScrapers {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Omit("created_by").Create(&c).Error; err != nil {
			return fmt.Errorf("error upserting config scraper: (id=%s): %w", c.ID, err)
		}
	}

	if len(req.ConfigItems) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.ConfigItems, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.ConfigItems, func(i models.ConfigItem, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigRelationships) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: models.ConfigRelationship{}.PKCols()}).CreateInBatches(req.ConfigRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.ConfigRelationships, func(i models.ConfigRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigComponentRelationships) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: models.ConfigComponentRelationship{}.PKCols()}).CreateInBatches(req.ConfigComponentRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.ConfigComponentRelationships, func(i models.ConfigComponentRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigChanges) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Omit("created_by").CreateInBatches(req.ConfigChanges, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.ConfigChanges, func(i models.ConfigChange, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigAnalysis) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Omit("created_by").CreateInBatches(req.ConfigAnalysis, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.ConfigAnalysis, func(i models.ConfigAnalysis, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.Checks) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.Checks, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.Checks, func(i models.Check, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.Artifacts) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.Artifacts, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting artifacts: %w", err)
		}
	}

	if len(req.JobHistory) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.JobHistory, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting job_history: %w", err)
		}
	}

	if len(req.ViewPanels) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.ViewPanels, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting view_panels: %w", err)
		}
	}

	for _, viewData := range req.GeneratedViews {
		if err := upsertViewData(ctx, viewData); err != nil {
			return fmt.Errorf("error upserting view_data: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: models.CheckStatus{}.PKCols()}).CreateInBatches(req.CheckStatuses, batchSize).Error; err != nil {
			return handleUpsertError(ctx, lo.Map(req.CheckStatuses, func(i models.CheckStatus, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	for i := range req.PlaybookActions {
		updates := map[string]any{
			"result":     req.PlaybookActions[i].Result,
			"start_time": req.PlaybookActions[i].StartTime,
			"end_time":   req.PlaybookActions[i].EndTime,
			"status":     req.PlaybookActions[i].Status,
			"error":      req.PlaybookActions[i].Error,
		}
		if err := db.Model(&models.PlaybookRunAction{}).Where("id = ?", req.PlaybookActions[i].ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("error updating playbook action [%s]: %w", req.PlaybookActions[i].ID, err)
		}

		if err := db.Exec("UPDATE playbook_runs SET status = ? WHERE id = (SELECT playbook_run_id FROM playbook_run_actions WHERE id = ?)", models.PlaybookRunStatusScheduled, req.PlaybookActions[i].ID).Error; err != nil {
			return fmt.Errorf("error updating playbook run [%s]  status to %s : %w", req.PlaybookActions[i].PlaybookRunID, models.PlaybookRunStatusScheduled, err)
		}
	}

	return nil
}

func handleUpsertError(ctx context.Context, items []models.ExtendedDBTable, err error) error {
	if !dutil.IsForeignKeyError(err) {
		return fmt.Errorf("error upserting: %w", err)
	}

	// If foreign key error, try inserting one by one and return the ones that fail
	var conflicted []string
	for _, item := range items {
		if err := ctx.DB().Clauses(clause.OnConflict{UpdateAll: true, Columns: item.PKCols()}).Omit("created_by").Create(item.Value()).Error; err != nil {
			if dutil.IsForeignKeyError(err) {
				conflicted = append(conflicted, item.PK())
			} else {
				return fmt.Errorf("error upserting config change (%s): %w", item.PK(), err)
			}
		}
	}

	conflicted = lo.Uniq(conflicted)
	return api.Errorf(api.ECONFLICT, ForeignKeyError).
		WithData(PushFKError{IDs: conflicted}).
		WithDebugInfo("foreign key error for %d items", len(conflicted))
}

func UpdateAgentLastSeen(ctx context.Context, id uuid.UUID) error {
	return ctx.DB().Model(&models.Agent{}).Where("id = ?", id).Update("last_seen", "NOW()").Error
}

func UpdateAgentLastReceived(ctx context.Context, id uuid.UUID) error {
	return ctx.DB().Model(&models.Agent{}).Where("id = ?", id).UpdateColumns(map[string]any{
		"last_received": gorm.Expr("NOW()"),
		"last_seen":     gorm.Expr("NOW()"),
	}).Error
}

// saveIndividuallyWithRetries saves the given records one by one and retries only on foreign key violation error.
func saveIndividuallyWithRetries[T models.DBTable](ctx context.Context, items []T, maxRetries int) error {
	var retries int
	for {
		var failed []T
		for _, c := range items {
			if err := ctx.DB().Clauses(clause.OnConflict{UpdateAll: true}).Omit("created_by").Create(&c).Error; err != nil {
				if dutil.IsForeignKeyError(err) {
					failed = append(failed, c)
				} else {
					return fmt.Errorf("error upserting %s (id=%s) : %w", c.TableName(), c.PK(), err)
				}
			}
		}

		if len(failed) == 0 {
			return nil
		}

		if retries > maxRetries {
			return api.Errorf(api.ECONFLICT, ForeignKeyError).
				WithData(PushFKError{IDs: lo.Map(failed, func(i T, _ int) string { return i.PK() })}).
				WithDebugInfo("foreign key error for %d items after %d retries", len(failed), retries)
		}

		items = failed
		retries++
		ctx.Tracef("retrying %d times to save %d items", retries, len(failed))
	}
}
