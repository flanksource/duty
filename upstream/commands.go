package upstream

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// agentOwnedRow holds, for each table agents push, the condition under which a row belongs to the agent given as ?.
// A row with an agent_id belongs to that agent; any other row belongs to the agent of its parent.
var agentOwnedRow = map[string]string{
	"topologies":      "topologies.agent_id = ?",
	"canaries":        "canaries.agent_id = ?",
	"checks":          "checks.agent_id = ?",
	"components":      "components.agent_id = ?",
	"config_scrapers": "config_scrapers.agent_id = ?",
	"config_items":    "config_items.agent_id = ?",
	"job_history":     "job_history.agent_id = ?",
	"view_panels":     "view_panels.agent_id = ?",

	"component_relationships":        "EXISTS (SELECT 1 FROM components WHERE components.id = component_relationships.component_id AND components.agent_id = ?)",
	"config_relationships":           "EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_relationships.config_id AND config_items.agent_id = ?)",
	"config_component_relationships": "EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_component_relationships.config_id AND config_items.agent_id = ?)",
	"check_config_relationships":     "EXISTS (SELECT 1 FROM checks WHERE checks.id = check_config_relationships.check_id AND checks.agent_id = ?)",
	"config_items_last_scraped_time": "EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_items_last_scraped_time.config_id AND config_items.agent_id = ?)",
	"checks_unlogged":                "EXISTS (SELECT 1 FROM checks WHERE checks.id = checks_unlogged.check_id AND checks.agent_id = ?)",
	"config_changes":                 "EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_changes.config_id AND config_items.agent_id = ?)",
	"config_analysis":                "EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_analysis.config_id AND config_items.agent_id = ?)",
	"check_statuses":                 "EXISTS (SELECT 1 FROM checks WHERE checks.id = check_statuses.check_id AND checks.agent_id = ?)",
	"artifacts": `EXISTS (SELECT 1 FROM checks WHERE checks.id = artifacts.check_id AND checks.agent_id = @agent)
		OR EXISTS (SELECT 1 FROM playbook_run_actions WHERE playbook_run_actions.id = artifacts.playbook_run_action_id AND playbook_run_actions.agent_id = @agent)
		OR EXISTS (SELECT 1 FROM config_changes JOIN config_items ON config_items.id = config_changes.config_id WHERE config_changes.id = artifacts.config_change_id AND config_items.agent_id = @agent)
		OR EXISTS (SELECT 1 FROM config_scrapers WHERE config_scrapers.id = artifacts.scraper_id AND config_scrapers.agent_id = @agent)
		OR EXISTS (SELECT 1 FROM job_history WHERE job_history.id = artifacts.job_history_id AND job_history.agent_id = @agent)`,
}

// ownedBy returns the condition under which a row of the table belongs to the agent.
// No row of a table without a condition belongs to the agent.
func ownedBy(table string, agentID uuid.UUID) clause.Expression {
	condition, ok := agentOwnedRow[table]
	if !ok {
		// Fail closed instead of panicking: a push into a table missing from agentOwnedRow
		// inserts new rows but never updates or deletes existing ones.
		return clause.Expr{SQL: "FALSE"}
	}
	if strings.Contains(condition, "@agent") {
		return clause.NamedExpr{SQL: condition, Vars: []any{sql.Named("agent", agentID)}}
	}
	return clause.Expr{SQL: condition, Vars: []any{agentID}}
}

// upsertOwnedBy upserts rows of the table, but only updates an existing row that belongs to the agent,
// so an agent can't overwrite the rows of the host or of another agent.
func upsertOwnedBy(table string, agentID uuid.UUID, columns ...clause.Column) clause.OnConflict {
	return clause.OnConflict{
		Columns:   columns,
		UpdateAll: true,
		Where:     clause.Where{Exprs: []clause.Expression{ownedBy(table, agentID)}},
	}
}

// DeleteOnUpstream deletes the given resources of the agent on the upstream.
// Resources that don't belong to the agent are left alone.
func DeleteOnUpstream(ctx context.Context, agentID uuid.UUID, req *PushData) error {
	db := ctx.DB()
	owned := func(table string) *gorm.DB { return db.Where(ownedBy(table, agentID)) }

	if len(req.ComponentRelationships) > 0 {
		if err := owned("component_relationships").Delete(req.ComponentRelationships).Error; err != nil {
			return fmt.Errorf("error deleting component_relationships: %w", err)
		}
	}

	if len(req.ConfigRelationships) > 0 {
		for _, rel := range req.ConfigRelationships {
			tx := owned("config_relationships").Where("config_id = ?", rel.ConfigID).
				Where("related_id = ?", rel.RelatedID).
				Where("relation = ?", rel.Relation)
			if rel.ScraperID == nil {
				tx = tx.Where("scraper_id IS NULL")
			} else {
				tx = tx.Where("scraper_id = ?", *rel.ScraperID)
			}
			if err := tx.Delete(&models.ConfigRelationship{}).Error; err != nil {
				return fmt.Errorf("error deleting config_relationships: %w", err)
			}
		}
	}

	if len(req.ConfigComponentRelationships) > 0 {
		if err := owned("config_component_relationships").Delete(req.ConfigComponentRelationships).Error; err != nil {
			return fmt.Errorf("error deleting config_component_relationships: %w: %+v", err, req.ConfigComponentRelationships)
		}
	}

	if len(req.CheckConfigRelationships) > 0 {
		if err := owned("check_config_relationships").Delete(req.CheckConfigRelationships).Error; err != nil {
			return fmt.Errorf("error deleting check_config_relationships: %w", err)
		}
	}

	if len(req.ConfigItemsLastScrapedTime) > 0 {
		if err := owned("config_items_last_scraped_time").Delete(req.ConfigItemsLastScrapedTime).Error; err != nil {
			return fmt.Errorf("error deleting config_items_last_scraped_time: %w", err)
		}
	}

	if len(req.ChecksUnlogged) > 0 {
		if err := owned("checks_unlogged").Delete(req.ChecksUnlogged).Error; err != nil {
			return fmt.Errorf("error deleting checks_unlogged: %w", err)
		}
	}

	if len(req.ConfigChanges) > 0 {
		if err := owned("config_changes").Delete(req.ConfigChanges).Error; err != nil {
			return fmt.Errorf("error deleting config changes: %w", err)
		}
	}

	if len(req.ConfigAnalysis) > 0 {
		if err := owned("config_analysis").Delete(req.ConfigAnalysis).Error; err != nil {
			return fmt.Errorf("error deleting config analysis: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		if err := owned("check_statuses").Delete(req.CheckStatuses).Error; err != nil {
			return fmt.Errorf("error deleting check_statuses: %w", err)
		}
	}

	if len(req.Components) > 0 {
		if err := owned("components").Delete(req.Components).Error; err != nil {
			logger.Errorf("error deleting components: %w", err)
		}
	}

	if len(req.ConfigItems) > 0 {
		if err := owned("config_items").Delete(req.ConfigItems).Error; err != nil {
			logger.Errorf("error deleting config items: %w", err)
		}
	}

	if len(req.Checks) > 0 {
		if err := owned("checks").Delete(req.Checks).Error; err != nil {
			return fmt.Errorf("error deleting checks: %w", err)
		}
	}

	if len(req.ConfigScrapers) > 0 {
		if err := owned("config_scrapers").Delete(req.ConfigScrapers).Error; err != nil {
			return fmt.Errorf("error deleting config scrapers: %w", err)
		}
	}

	if len(req.Canaries) > 0 {
		if err := owned("canaries").Delete(req.Canaries).Error; err != nil {
			return fmt.Errorf("error deleting canaries: %w", err)
		}
	}

	if len(req.Topologies) > 0 {
		if err := owned("topologies").Delete(req.Topologies).Error; err != nil {
			return fmt.Errorf("error deleting topologies: %w", err)
		}
	}

	if len(req.JobHistory) > 0 {
		if err := owned("job_history").Delete(req.JobHistory).Error; err != nil {
			return fmt.Errorf("error deleting job_history: %w", err)
		}
	}

	if len(req.ViewPanels) > 0 {
		if err := owned("view_panels").Delete(req.ViewPanels).Error; err != nil {
			return fmt.Errorf("error deleting view_panels: %w", err)
		}
	}

	if len(req.GeneratedViews) > 0 {
		for _, viewData := range req.GeneratedViews {
			if err := deleteViewData(ctx, agentID, viewData); err != nil {
				return fmt.Errorf("error deleting view_data: %w", err)
			}
		}
	}

	return nil
}

// InsertUpstreamMsg saves the resources the agent pushed on the upstream.
// Resources that already exist and don't belong to the agent aren't updated.
func InsertUpstreamMsg(ctx context.Context, agentID uuid.UUID, req *PushData) error {
	batchSize := 100
	db := ctx.DB()
	for _, c := range req.Topologies {
		if err := db.Clauses(upsertOwnedBy("topologies", agentID)).Omit("created_by").Create(&c).Error; err != nil {
			return fmt.Errorf("error upserting topology: (id=%s): %w", c.ID, err)
		}
	}

	for _, c := range req.Canaries {
		if err := db.Clauses(upsertOwnedBy("canaries", agentID)).Omit("created_by").Create(&c).Error; err != nil {
			return fmt.Errorf("error upserting canaries: (id=%s): %w", c.ID, err)
		}
	}

	// components are inserted one by one, instead of in a batch, because of the foreign key constraint with itself.
	if err := saveIndividuallyWithRetries(ctx, agentID, req.Components, saveRetries); err != nil {
		return fmt.Errorf("error upserting components: %w", err)
	}

	if len(req.ComponentRelationships) > 0 {
		if err := db.Clauses(upsertOwnedBy("component_relationships", agentID, models.ComponentRelationship{}.PKCols()...)).CreateInBatches(req.ComponentRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ComponentRelationships, func(i models.ComponentRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	for _, c := range req.ConfigScrapers {
		if err := db.Clauses(upsertOwnedBy("config_scrapers", agentID)).Omit("created_by").Create(&c).Error; err != nil {
			return fmt.Errorf("error upserting config scraper: (id=%s): %w", c.ID, err)
		}
	}

	if len(req.ConfigItems) > 0 {
		if err := db.Clauses(upsertOwnedBy("config_items", agentID)).CreateInBatches(req.ConfigItems, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ConfigItems, func(i models.ConfigItem, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigRelationships) > 0 {
		if err := db.Clauses(upsertOwnedBy("config_relationships", agentID, models.ConfigRelationship{}.PKCols()...)).CreateInBatches(req.ConfigRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ConfigRelationships, func(i models.ConfigRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigComponentRelationships) > 0 {
		if err := db.Clauses(upsertOwnedBy("config_component_relationships", agentID, models.ConfigComponentRelationship{}.PKCols()...)).CreateInBatches(req.ConfigComponentRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ConfigComponentRelationships, func(i models.ConfigComponentRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.CheckConfigRelationships) > 0 {
		if err := db.Clauses(upsertOwnedBy("check_config_relationships", agentID, models.CheckConfigRelationship{}.PKCols()...)).CreateInBatches(req.CheckConfigRelationships, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.CheckConfigRelationships, func(i models.CheckConfigRelationship, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigItemsLastScrapedTime) > 0 {
		if err := db.Clauses(upsertOwnedBy("config_items_last_scraped_time", agentID, clause.Column{Name: "config_id"})).CreateInBatches(req.ConfigItemsLastScrapedTime, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ConfigItemsLastScrapedTime, func(i models.ConfigItemLastScrapedTime, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ChecksUnlogged) > 0 {
		if err := db.Clauses(upsertOwnedBy("checks_unlogged", agentID, clause.Column{Name: "check_id"})).CreateInBatches(req.ChecksUnlogged, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting checks_unlogged: %w", err)
		}
	}

	if len(req.ConfigChanges) > 0 {
		if err := db.Clauses(upsertOwnedBy("config_changes", agentID)).Omit("created_by").CreateInBatches(req.ConfigChanges, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ConfigChanges, func(i models.ConfigChange, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.ConfigAnalysis) > 0 {
		if err := db.Clauses(upsertOwnedBy("config_analysis", agentID)).Omit("created_by").CreateInBatches(req.ConfigAnalysis, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.ConfigAnalysis, func(i models.ConfigAnalysis, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.Checks) > 0 {
		if err := db.Clauses(upsertOwnedBy("checks", agentID)).CreateInBatches(req.Checks, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.Checks, func(i models.Check, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	if len(req.Artifacts) > 0 {
		if err := db.Clauses(upsertOwnedBy("artifacts", agentID)).CreateInBatches(req.Artifacts, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting artifacts: %w", err)
		}
	}

	if len(req.JobHistory) > 0 {
		if err := db.Clauses(upsertOwnedBy("job_history", agentID)).CreateInBatches(req.JobHistory, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting job_history: %w", err)
		}
	}

	if len(req.ViewPanels) > 0 {
		if err := db.Clauses(upsertOwnedBy("view_panels", agentID)).CreateInBatches(req.ViewPanels, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting view_panels: %w", err)
		}
	}

	for _, viewData := range req.GeneratedViews {
		if err := upsertViewData(ctx, viewData); err != nil {
			return fmt.Errorf("error upserting view_data: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		if err := db.Clauses(upsertOwnedBy("check_statuses", agentID, models.CheckStatus{}.PKCols()...)).CreateInBatches(req.CheckStatuses, batchSize).Error; err != nil {
			return handleUpsertError(ctx, agentID, lo.Map(req.CheckStatuses, func(i models.CheckStatus, _ int) models.ExtendedDBTable { return i }), err)
		}
	}

	for _, reported := range req.PlaybookActions {
		var action models.PlaybookRunAction
		if err := db.Select("id", "playbook_run_id").Where("id = ? AND agent_id = ?", reported.ID, agentID).Limit(1).Find(&action).Error; err != nil {
			return fmt.Errorf("error fetching playbook action [%s]: %w", reported.ID, err)
		} else if action.ID == uuid.Nil {
			continue
		}

		if err := action.SaveAgentResult(db, reported); err != nil {
			if errors.Is(err, models.ErrPlaybookTransition) {
				// Usually a result the agent pushed again. The action already ended, so there's nothing to apply.
				ctx.Warnf("ignoring pushed playbook action: %v", err)
				continue
			}
			return fmt.Errorf("error saving pushed playbook action [%s]: %w", reported.ID, err)
		}
	}

	return nil
}

func handleUpsertError(ctx context.Context, agentID uuid.UUID, items []models.ExtendedDBTable, err error) error {
	if !dutil.IsForeignKeyError(err) {
		return fmt.Errorf("error upserting: %w", err)
	}

	// If foreign key error, try inserting one by one and return the ones that fail
	var conflicted []string
	for _, item := range items {
		if err := ctx.DB().Clauses(upsertOwnedBy(item.TableName(), agentID, item.PKCols()...)).Omit("created_by").Create(item.Value()).Error; err != nil {
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
func saveIndividuallyWithRetries[T models.DBTable](ctx context.Context, agentID uuid.UUID, items []T, maxRetries int) error {
	var retries int
	for {
		var failed []T
		for _, c := range items {
			if err := ctx.DB().Clauses(upsertOwnedBy(c.TableName(), agentID)).Omit("created_by").Create(&c).Error; err != nil {
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
