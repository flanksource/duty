package view

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/Masterminds/squirrel"
	"github.com/flanksource/commons/logger"
	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

const ReservedColumnAttributes = "__row__attributes"

// Row represents a single row of data mapped to view columns
type Row []any

func GetViewColumnDefs(ctx context.Context, namespace, name string) (ViewColumnDefList, error) {
	var view models.View
	err := ctx.DB().Where("namespace = ? AND name = ?", namespace, name).First(&view).Error
	if err != nil {
		return nil, err
	}

	var spec struct {
		Columns []ColumnDef `json:"columns"`
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
	return applyViewTableSchema(ctx, table, columns)
}

func applyViewTableSchema(ctx context.Context, tableName string, columns ViewColumnDefList) error {
	primaryKeys := columns.PrimaryKey()
	if len(primaryKeys) == 0 {
		return fmt.Errorf("no primary key columns found in view table definition")
	}

	client, err := sqlclient.Open(ctx, ctx.ConnectionString())
	if err != nil {
		return fmt.Errorf("failed to open SQL client: %w", err)
	}
	defer client.Close()

	currentState, err := client.InspectSchema(ctx, api.DefaultConfig.Schema, &schema.InspectOptions{Tables: []string{tableName}})
	if err != nil {
		return fmt.Errorf("failed to inspect schema for table %s: %w", tableName, err)
	}

	desiredState := createTableSchema(tableName, columns, currentState)

	var changes []schema.Change
	if len(currentState.Tables) == 0 {
		changes = []schema.Change{&schema.AddTable{T: desiredState}}
	} else {
		tableDiff, err := client.SchemaDiff(currentState, &schema.Schema{
			Name:   api.DefaultConfig.Schema,
			Tables: []*schema.Table{desiredState},
		},
			schema.DiffSkipChanges(
				&schema.DropTable{}, &schema.DropSchema{}, &schema.DropObject{},
			),
		)
		if err != nil {
			return fmt.Errorf("failed to compute table diff: %w", err)
		}
		changes = tableDiff
	}

	if len(changes) > 0 {
		if err := client.ApplyChanges(ctx, changes); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}
	}

	return nil
}

func createTableSchema(tableName string, columns ViewColumnDefList, currentSchema *schema.Schema) *schema.Table {
	table := &schema.Table{
		Name:   tableName,
		Schema: currentSchema,
	}

	for _, col := range columns {
		column := &schema.Column{
			Name: col.Name,
			Type: &schema.ColumnType{
				Type: getAtlasType(col.Type),
				Null: true, // Assume all columns are nullable (except primary key)
			},
		}

		if col.PrimaryKey {
			column.Type.Null = false
		}

		table.Columns = append(table.Columns, column)
	}

	// Always add this column to keep track of the attributes of the columns in a row.
	// Example: column.url = "https://flanksource.com"
	table.Columns = append(table.Columns, &schema.Column{
		Name: ReservedColumnAttributes,
		Type: &schema.ColumnType{
			Type: &schema.JSONType{T: "jsonb"},
			Null: true,
		},
	})

	// Add columns used for upstream reconciliation
	table.Columns = append(table.Columns, &schema.Column{
		Name: "agent_id",
		Type: &schema.ColumnType{
			Type: &schema.UUIDType{T: "uuid"},
		},
		Default: &schema.RawExpr{
			X: fmt.Sprintf("'%s'::uuid", uuid.Nil),
		},
	}, &schema.Column{
		Name: "is_pushed",
		Type: &schema.ColumnType{
			Type: &schema.BoolType{T: "boolean"},
		},
		Default: &schema.RawExpr{
			X: "false",
		},
	})

	primaryKeys := columns.PrimaryKey()
	var pkColumns []*schema.Column
	for _, col := range table.Columns {
		if lo.Contains(primaryKeys, col.Name) {
			pkColumns = append(pkColumns, col)
		}
	}

	if len(pkColumns) > 0 {
		table.PrimaryKey = &schema.Index{
			Name:   fmt.Sprintf("%s_pkey", tableName),
			Unique: true,
			Table:  table,
		}

		for _, col := range pkColumns {
			table.PrimaryKey.Parts = append(table.PrimaryKey.Parts, &schema.IndexPart{
				C: col,
			})
		}
	}

	return table
}

func getAtlasType(colType ColumnType) schema.Type {
	switch colType {
	case ColumnTypeString:
		return &schema.StringType{T: "text"}
	case ColumnTypeNumber:
		return &schema.DecimalType{T: "numeric"}
	case ColumnTypeBoolean:
		return &schema.BoolType{T: "boolean"}
	case ColumnTypeDateTime:
		return &schema.TimeType{T: "timestamptz"}
	case ColumnTypeDuration:
		return &schema.IntegerType{T: "bigint"}
	case ColumnTypeHealth:
		return &schema.StringType{T: "text"}
	case ColumnTypeStatus:
		return &schema.StringType{T: "text"}
	case ColumnTypeGauge:
		return &schema.JSONType{T: "jsonb"}
	case ColumnTypeBytes:
		return &schema.StringType{T: "text"} // stored as text due to values like "250Mi"
	case ColumnTypeMillicore:
		return &schema.StringType{T: "text"}
	case ColumnTypeAttributes:
		return &schema.JSONType{T: "jsonb"}
	default:
		return &schema.StringType{T: "text"}
	}
}

func ReadViewTable(ctx context.Context, columnDef ViewColumnDefList, table string) ([]Row, error) {
	columns := columnDef.QuotedColumns()

	rows, err := ctx.DB().Select(strings.Join(columns, ", ")).Table(table).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to read view table (%s): %w", table, err)
	}
	defer rows.Close()

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

	return convertViewRecordsToNativeTypes(viewRows, columnDef), nil
}

// convertViewRecordsToNativeTypes converts view cell to native go types
func convertViewRecordsToNativeTypes(viewRows []Row, columnDef ViewColumnDefList) []Row {
	for _, viewRow := range viewRows {
		for i, colDef := range columnDef {
			if i >= len(viewRow) {
				continue
			}

			if viewRow[i] == nil {
				continue
			}

			switch colDef.Type {
			case ColumnTypeGauge:
				if raw, ok := viewRow[i].([]uint8); ok {
					viewRow[i] = json.RawMessage(raw)
				}

			case ColumnTypeAttributes:
				if raw, ok := viewRow[i].([]uint8); ok {
					viewRow[i] = json.RawMessage(raw)
				}

			case ColumnTypeDuration:
				switch v := viewRow[i].(type) {
				case int:
					viewRow[i] = time.Duration(v)
				case int32:
					viewRow[i] = time.Duration(v)
				case int64:
					viewRow[i] = time.Duration(v)
				case float64:
					viewRow[i] = time.Duration(int64(v))
				default:
					logger.Warnf("convertViewRecordsToNativeTypes: unknown duration type: %T", v)
				}

			case ColumnTypeDateTime:
				switch v := viewRow[i].(type) {
				case time.Time:
					viewRow[i] = v
				case string:
					parsed, err := time.Parse(time.RFC3339, v)
					if err != nil {
						logger.Warnf("convertViewRecordsToNativeTypes: failed to parse datetime: %v", err)
					}
					viewRow[i] = parsed
				default:
					logger.Warnf("convertViewRecordsToNativeTypes: unknown datetime type: %T", v)
				}
			}
		}
	}

	return viewRows
}

func InsertViewRows(ctx context.Context, table string, columns ViewColumnDefList, rows []Row) error {
	if len(rows) == 0 {
		return ctx.DB().Exec(fmt.Sprintf("DELETE FROM %s", pq.QuoteIdentifier(table))).Error
	}

	quotedColumns := columns.QuotedColumns()
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert(table).Columns(quotedColumns...)
	for _, row := range rows {
		insertBuilder = insertBuilder.Values(row...)
	}

	pkColumns := columns.PrimaryKey()
	quotedPrimaryKeys := lo.Map(pkColumns, func(col string, _ int) string {
		return pq.QuoteIdentifier(col)
	})

	conflictCols := strings.Join(quotedPrimaryKeys, ", ")
	var updateClauses []string
	for _, col := range columns {
		if !col.PrimaryKey {
			quotedCol := pq.QuoteIdentifier(col.Name)
			updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", quotedCol, quotedCol))
		}
	}

	upsertBuilder := insertBuilder.Suffix(
		fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET %s RETURNING %s",
			conflictCols,
			strings.Join(updateClauses, ", "),
			strings.Join(pkColumns, ", "),
		),
	)

	upsertSQL, args, err := upsertBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build upsert query: %w", err)
	}

	var pkEq []string
	for _, pk := range pkColumns {
		q := pq.QuoteIdentifier(pk)
		pkEq = append(pkEq, fmt.Sprintf("t.%s = upsert.%s", q, q))
	}

	finalSQL := fmt.Sprintf(`
		WITH upsert AS (
			%s
		)
		DELETE FROM %s AS t
		WHERE NOT EXISTS (
			SELECT 1 FROM upsert
			WHERE %s
		)`,
		upsertSQL,
		pq.QuoteIdentifier(table),
		strings.Join(pkEq, " AND "),
	)

	return ctx.DB().Exec(finalSQL, args...).Error
}
