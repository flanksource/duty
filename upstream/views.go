package upstream

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// deleteViewData deletes records from dynamic view_* tables
func deleteViewData(ctx context.Context, viewData []models.GeneratedViewTable) error {
	if len(viewData) == 0 {
		return nil
	}

	// Get agent ID for table name suffix
	agent := ctx.Agent()
	if agent == nil {
		return fmt.Errorf("agent context not found")
	}

	// Group by table name
	tableGroups := make(map[string][]models.GeneratedViewTable)
	for _, data := range viewData {
		// Create agent-specific table name
		agentTableName := fmt.Sprintf("%s_%s", data.ViewTableName, agent.ID.String())
		tableGroups[agentTableName] = append(tableGroups[agentTableName], data)
	}

	// Delete from each table
	for agentTableName, records := range tableGroups {
		if !ctx.DB().Migrator().HasTable(agentTableName) {
			continue
		}

		// Create a basic delete query
		var conditions []string
		var args []any
		for _, record := range records {
			// Use a simple condition based on available data
			if id, ok := record.Row["id"]; ok {
				conditions = append(conditions, "id = ?")
				args = append(args, id)
			}
		}

		if len(conditions) > 0 {
			query := fmt.Sprintf("DELETE FROM %s WHERE %s", agentTableName, strings.Join(conditions, " OR "))
			if err := ctx.DB().Exec(query, args...).Error; err != nil {
				return fmt.Errorf("error deleting from %s: %w", agentTableName, err)
			}
		}
	}

	return nil
}

// upsertViewData handles upserting records to dynamic view_* tables
func upsertViewData(ctx context.Context, viewData []models.GeneratedViewTable) error {
	if len(viewData) == 0 {
		return nil
	}

	table := viewData[0].ViewTableName

	columns := make([]string, 0, len(viewData[0].Row))
	for key := range viewData[0].Row {
		columns = append(columns, key)
	}

	insertBuilder := squirrel.Insert(table).PlaceholderFormat(squirrel.Dollar).Columns(columns...)

	for _, record := range viewData {
		values := make([]any, 0, len(columns))
		for _, col := range columns {
			values = append(values, record.Row[col])
		}
		insertBuilder = insertBuilder.Values(values...)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("error building batch insert query: %w", err)
	}

	if err := ctx.DB().Exec(query, args...).Error; err != nil {
		return fmt.Errorf("error batch upserting to %s: %w", table, err)
	}

	return nil
}
