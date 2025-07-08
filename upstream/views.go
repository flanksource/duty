package upstream

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func deleteViewData(ctx context.Context, records []models.GeneratedViewTable) error {
	if len(records) == 0 {
		return nil
	}

	table := records[0].ViewTableName
	if !strings.HasPrefix(table, "view_") {
		return fmt.Errorf("table %s is not a view generated table", table)
	}

	deleteBuilder := squirrel.Delete(table).PlaceholderFormat(squirrel.Dollar)

	for _, record := range records {
		if len(record.PrimaryKey) == 0 {
			return fmt.Errorf("primary key not found for table: %s", table)
		} else if len(record.PrimaryKey) > 1 {
			return fmt.Errorf("multiple primary keys found for table: %s", table)
		}

		deleteBuilder = deleteBuilder.Where(squirrel.Eq{
			record.PrimaryKey[0]: record.Row[record.PrimaryKey[0]],
		})
	}

	query, args, err := deleteBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("error building delete query: %w", err)
	}

	if err := ctx.DB().Exec(query, args...).Error; err != nil {
		return fmt.Errorf("error deleting from %s: %w", table, err)
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
