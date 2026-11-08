package upstream

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

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
	"topologies":           "topologies.agent_id = ?",
	"canaries":             "canaries.agent_id = ?",
	"checks":               "checks.agent_id = ?",
	"components":           "components.agent_id = ?",
	"config_scrapers":      "config_scrapers.agent_id = ?",
	"config_items":         "config_items.agent_id = ?",
	"job_history":          "job_history.agent_id = ?",
	"view_panels":          "view_panels.agent_id = ?",
	"playbook_run_actions": "playbook_run_actions.agent_id = ?",

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

// dropForeign drops the rows that reference a row of the parent table belonging to another agent, so an agent
// can't attach rows to the resources of the host or of another agent.
// A reference to a row that doesn't exist yet is kept: the push then fails with a foreign key error and is retried.
func dropForeign[T any](ctx context.Context, agentID uuid.UUID, rows []T, parent string, ref func(T) string) ([]T, error) {
	var ids []uuid.UUID
	for _, row := range rows {
		if id, err := uuid.Parse(ref(row)); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return rows, nil
	}

	var foreign []uuid.UUID
	if err := ctx.DB().Table(parent).Where("id IN ?", lo.Uniq(ids)).Not(ownedBy(parent, agentID)).Pluck("id", &foreign).Error; err != nil {
		return nil, fmt.Errorf("error looking up the owners of %s: %w", parent, err)
	}
	if len(foreign) == 0 {
		return rows, nil
	}

	ctx.Warnf("agent %s pushed rows referencing %d %s of another agent", agentID, len(foreign), parent)
	return lo.Filter(rows, func(row T, _ int) bool {
		id, err := uuid.Parse(ref(row))
		return err != nil || !lo.Contains(foreign, id)
	}), nil
}

// dropForeignRows drops the pushed rows that reference the resources of the host or of another agent.
func dropForeignRows(ctx context.Context, agentID uuid.UUID, req *PushData) error {
	str := func(id *uuid.UUID) string {
		if id == nil {
			return ""
		}
		return id.String()
	}

	var err error
	if req.ComponentRelationships, err = dropForeign(ctx, agentID, req.ComponentRelationships, "components", func(r models.ComponentRelationship) string { return r.ComponentID.String() }); err != nil {
		return err
	}
	if req.ConfigRelationships, err = dropForeign(ctx, agentID, req.ConfigRelationships, "config_items", func(r models.ConfigRelationship) string { return r.ConfigID }); err != nil {
		return err
	}
	if req.ConfigComponentRelationships, err = dropForeign(ctx, agentID, req.ConfigComponentRelationships, "config_items", func(r models.ConfigComponentRelationship) string { return r.ConfigID.String() }); err != nil {
		return err
	}
	if req.CheckConfigRelationships, err = dropForeign(ctx, agentID, req.CheckConfigRelationships, "checks", func(r models.CheckConfigRelationship) string { return r.CheckID.String() }); err != nil {
		return err
	}
	if req.ConfigItemsLastScrapedTime, err = dropForeign(ctx, agentID, req.ConfigItemsLastScrapedTime, "config_items", func(r models.ConfigItemLastScrapedTime) string { return r.ConfigID.String() }); err != nil {
		return err
	}
	if req.ChecksUnlogged, err = dropForeign(ctx, agentID, req.ChecksUnlogged, "checks", func(r models.ChecksUnlogged) string { return r.CheckID.String() }); err != nil {
		return err
	}
	if req.ConfigChanges, err = dropForeign(ctx, agentID, req.ConfigChanges, "config_items", func(r models.ConfigChange) string { return r.ConfigID }); err != nil {
		return err
	}
	if req.ConfigAnalysis, err = dropForeign(ctx, agentID, req.ConfigAnalysis, "config_items", func(r models.ConfigAnalysis) string { return r.ConfigID.String() }); err != nil {
		return err
	}
	if req.CheckStatuses, err = dropForeign(ctx, agentID, req.CheckStatuses, "checks", func(r models.CheckStatus) string { return r.CheckID.String() }); err != nil {
		return err
	}

	// A job history of another agent is dropped, not only left unchanged by the upsert: the push handler adds the
	// pushed histories to the agent's status ring, which deletes the ones it evicts.
	if req.JobHistory, err = dropForeign(ctx, agentID, req.JobHistory, "job_history", func(r models.JobHistory) string { return r.ID.String() }); err != nil {
		return err
	}

	// An artifact must not reference any resource of another agent, even when another of its references is the agent's.
	artifactParents := map[string]func(models.Artifact) string{
		"checks":               func(a models.Artifact) string { return str(a.CheckID) },
		"playbook_run_actions": func(a models.Artifact) string { return str(a.PlaybookRunActionID) },
		"config_changes":       func(a models.Artifact) string { return str(a.ConfigChangeID) },
		"config_scrapers":      func(a models.Artifact) string { return str(a.ScraperID) },
		"job_history":          func(a models.Artifact) string { return str(a.JobHistoryID) },
	}
	for parent, ref := range artifactParents {
		if req.Artifacts, err = dropForeign(ctx, agentID, req.Artifacts, parent, ref); err != nil {
			return err
		}
	}

	return nil
}

// DeleteOnUpstream deletes the given resources of the agent on the upstream.
// Resources that don't belong to the agent are left alone.
func DeleteOnUpstream(ctx context.Context, agentID uuid.UUID, req *PushData) error {
	db := ctx.DB()
	owned := func(table string) *gorm.DB { return db.Where(ownedBy(table, agentID)) }

	// deleteOwned deletes the rows by their primary key. Rows without one are skipped: GORM leaves them out of
	// the WHERE clause, so the ownership condition alone would delete every row the agent owns.
	deleteOwned := func(table string, rows any) error {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(rows); err != nil {
			return err
		}
		if _, keys := schema.GetIdentityFieldValuesMap(ctx, reflect.ValueOf(rows), stmt.Schema.PrimaryFields); len(keys) == 0 {
			return nil
		}
		return owned(table).Delete(rows).Error
	}

	if len(req.ComponentRelationships) > 0 {
		if err := deleteOwned("component_relationships", req.ComponentRelationships); err != nil {
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
		if err := deleteOwned("config_component_relationships", req.ConfigComponentRelationships); err != nil {
			return fmt.Errorf("error deleting config_component_relationships: %w: %+v", err, req.ConfigComponentRelationships)
		}
	}

	if len(req.CheckConfigRelationships) > 0 {
		if err := deleteOwned("check_config_relationships", req.CheckConfigRelationships); err != nil {
			return fmt.Errorf("error deleting check_config_relationships: %w", err)
		}
	}

	if len(req.ConfigItemsLastScrapedTime) > 0 {
		if err := deleteOwned("config_items_last_scraped_time", req.ConfigItemsLastScrapedTime); err != nil {
			return fmt.Errorf("error deleting config_items_last_scraped_time: %w", err)
		}
	}

	if len(req.ChecksUnlogged) > 0 {
		if err := deleteOwned("checks_unlogged", req.ChecksUnlogged); err != nil {
			return fmt.Errorf("error deleting checks_unlogged: %w", err)
		}
	}

	if len(req.ConfigChanges) > 0 {
		if err := deleteOwned("config_changes", req.ConfigChanges); err != nil {
			return fmt.Errorf("error deleting config changes: %w", err)
		}
	}

	if len(req.ConfigAnalysis) > 0 {
		if err := deleteOwned("config_analysis", req.ConfigAnalysis); err != nil {
			return fmt.Errorf("error deleting config analysis: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		if err := deleteOwned("check_statuses", req.CheckStatuses); err != nil {
			return fmt.Errorf("error deleting check_statuses: %w", err)
		}
	}

	if len(req.Components) > 0 {
		if err := deleteOwned("components", req.Components); err != nil {
			logger.Errorf("error deleting components: %w", err)
		}
	}

	if len(req.ConfigItems) > 0 {
		if err := deleteOwned("config_items", req.ConfigItems); err != nil {
			logger.Errorf("error deleting config items: %w", err)
		}
	}

	if len(req.Checks) > 0 {
		if err := deleteOwned("checks", req.Checks); err != nil {
			return fmt.Errorf("error deleting checks: %w", err)
		}
	}

	if len(req.ConfigScrapers) > 0 {
		if err := deleteOwned("config_scrapers", req.ConfigScrapers); err != nil {
			return fmt.Errorf("error deleting config scrapers: %w", err)
		}
	}

	if len(req.Canaries) > 0 {
		if err := deleteOwned("canaries", req.Canaries); err != nil {
			return fmt.Errorf("error deleting canaries: %w", err)
		}
	}

	if len(req.Topologies) > 0 {
		if err := deleteOwned("topologies", req.Topologies); err != nil {
			return fmt.Errorf("error deleting topologies: %w", err)
		}
	}

	if len(req.JobHistory) > 0 {
		if err := deleteOwned("job_history", req.JobHistory); err != nil {
			return fmt.Errorf("error deleting job_history: %w", err)
		}
	}

	if len(req.ViewPanels) > 0 {
		if err := deleteOwned("view_panels", req.ViewPanels); err != nil {
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
// Resources that already exist and don't belong to the agent aren't updated, and rows referencing them are dropped.
func InsertUpstreamMsg(ctx context.Context, agentID uuid.UUID, req *PushData) error {
	if err := dropForeignRows(ctx, agentID, req); err != nil {
		return err
	}

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
