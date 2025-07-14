package view

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// Row represents a single row of data mapped to view columns
type Row []any

func GetViewColumnDefs(ctx context.Context, namespace, name string) (ViewColumnDefList, error) {
	var view models.View
	err := ctx.DB().Where("namespace = ? AND name = ?", namespace, name).First(&view).Error
	if err != nil {
		return nil, err
	}

	var spec struct {
		Columns []ViewColumnDef `json:"columns"`
	}

	err = json.Unmarshal(view.Spec, &spec)
	if err != nil {
		return nil, err
	}

	return spec.Columns, nil
}

func GetAllViews(ctx context.Context) ([]models.View, error) {
	var views []models.View
	if err := ctx.DB().Where("deleted_at IS NULL").Find(&views).Error; err != nil {
		return nil, err
	}

	return views, nil
}

func CreateViewTable(ctx context.Context, table string, columns ViewColumnDefList) error {
	if ctx.DB().Migrator().HasTable(table) {
		return nil
	}

	primaryKeys := columns.PrimaryKey()
	if len(primaryKeys) == 0 {
		return fmt.Errorf("no primary key columns found in view table definition")
	}

	var columnDefs []string
	for _, col := range columns {
		colDef := fmt.Sprintf("%s %s", col.Name, getPostgresType(col.Type))
		columnDefs = append(columnDefs, colDef)
	}

	columnDefs = append(columnDefs, "agent_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid")
	columnDefs = append(columnDefs, "is_pushed BOOLEAN DEFAULT FALSE")

	primaryKeyConstraint := fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", "))
	columnDefs = append(columnDefs, primaryKeyConstraint)

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, strings.Join(columnDefs, ", "))
	return ctx.DB().Exec(sql).Error
}

func getPostgresType(colType ColumnType) string {
	switch colType {
	case ColumnTypeString:
		return "TEXT"
	case ColumnTypeNumber:
		return "NUMERIC"
	case ColumnTypeBoolean:
		return "BOOLEAN"
	case ColumnTypeDateTime:
		return "TIMESTAMP WITH TIME ZONE"
	case ColumnTypeDuration:
		return "BIGINT"
	case ColumnTypeHealth:
		return "TEXT"
	case ColumnTypeStatus:
		return "TEXT"
	case ColumnTypeGauge:
		return "JSONB"
	default:
		return "TEXT"
	}
}

func ReadViewTable(ctx context.Context, table string) ([]Row, error) {
	rows, err := ctx.DB().Select("*").Table(table).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to read view table (%s): %w", table, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for view table (%s): %w", table, err)
	}

	var viewRows []Row
	for rows.Next() {
		viewRow := make(Row, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range viewRow {
			valuePtrs[i] = &viewRow[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		viewRows = append(viewRows, viewRow)
	}

	return viewRows, nil
}

func InsertViewRows(ctx context.Context, table string, columns []ViewColumnDef, rows []Row) error {
	if err := ctx.DB().Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
		return fmt.Errorf("failed to clear existing data: %w", err)
	}

	if len(rows) == 0 {
		return nil
	}

	var colNames []string
	for _, col := range columns {
		colNames = append(colNames, col.Name)
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert(table).Columns(colNames...)
	for _, row := range rows {
		insertBuilder = insertBuilder.Values(row...)
	}

	sql, args, err := insertBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	return ctx.DB().Exec(sql, args...).Error
}
