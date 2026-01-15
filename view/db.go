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
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

const (
	ReservedColumnAttributes = "__row__attributes"
	ReservedColumnGrants     = "__grants"
)

// Row represents a single row of data mapped to view columns
type Row []any

func GetViewColumnDefs(ctx context.Context, namespace, name string) (ViewColumnDefList, error) {
	var view models.View
	err := ctx.DB().Where("namespace = ? AND name = ?", namespace, name).Where("deleted_at IS NULL").First(&view).Error
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
			// The schema migration has failed. This can be due to
			// - incompatible/breaking changes (e.g., changing primary key)
			// - or due to invalid schema.
			// If it's the first case, we need to drop and recreate the table.
			// Else, the view spec needs to be fixed by the user.

			currentState, inspectErr := client.InspectSchema(ctx, api.DefaultConfig.Schema, &schema.InspectOptions{Tables: []string{tableName}})
			if inspectErr != nil {
				return fmt.Errorf("failed to re-inspect schema for table %s: %w (original error: %v)", tableName, inspectErr, err)
			} else if len(currentState.Tables) == 0 {
				// The table doesn't even exist. There's no point in re-trying.
				return fmt.Errorf("failed to recreate table %s: %w", tableName, err)
			}

			ctx.Logger.Warnf("View table migration failed for %s, dropping and recreating (data will be lost): %v", tableName, err)
			changesToRecreate := []schema.Change{
				&schema.DropTable{T: currentState.Tables[0]},
				&schema.AddTable{T: desiredState},
			}
			if err := client.ApplyChanges(ctx, changesToRecreate); err != nil {
				return fmt.Errorf("failed to drop and recreate table %s: %w", tableName, err)
			}
		}
	}

	// Apply RLS policy to enforce grants
	// (Re)apply RLS and Policy on first table creation or on schema changes
	if len(changes) > 0 {
		if err := ensureViewRLSPolicy(ctx, tableName); err != nil {
			return fmt.Errorf("failed to apply RLS policy: %w", err)
		}
	}

	return nil
}

func ensureViewRLSPolicy(ctx context.Context, tableName string) error {
	// Enable RLS on table
	if err := ctx.DB().Exec("ALTER TABLE " + pq.QuoteIdentifier(tableName) + " ENABLE ROW LEVEL SECURITY").Error; err != nil {
		return fmt.Errorf("failed to enable RLS: %w", err)
	}

	// Drop existing policy if present
	if err := ctx.DB().
		Exec("DROP POLICY IF EXISTS view_grants_policy ON " + pq.QuoteIdentifier(tableName)).
		Error; err != nil {
		return fmt.Errorf("failed to drop existing RLS policy: %w", err)
	}

	// Create the grants policy
	policy := fmt.Sprintf(`
		CREATE POLICY view_grants_policy ON %s
			FOR ALL TO postgrest_api, postgrest_anon
			USING (
				check_view_grants(__grants)
			)
	`, pq.QuoteIdentifier(tableName))

	if err := ctx.DB().Exec(policy).Error; err != nil {
		return fmt.Errorf("failed to create RLS policy: %w", err)
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

	// Add grants column for row-level access control
	table.Columns = append(table.Columns, &schema.Column{
		Name: ReservedColumnGrants,
		Type: &schema.ColumnType{
			Type: &schema.JSONType{T: "jsonb"},
			Null: true,
		},
	})

	// Add request fingerprint column for cache differentiation
	table.Columns = append(table.Columns, &schema.Column{
		Name: "request_fingerprint",
		Type: &schema.ColumnType{
			Type: &schema.StringType{T: "text"},
			Null: false,
		},
		Default: &schema.RawExpr{
			X: "''",
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
		return &schema.FloatType{T: "float"}
	case ColumnTypeBytes:
		return &schema.IntegerType{T: "bigint"}
	case ColumnTypeMillicore:
		return &schema.FloatType{T: "float"}
	case ColumnTypeAttributes:
		return &schema.JSONType{T: "jsonb"}
	case ColumnTypeGrants:
		return &schema.JSONType{T: "jsonb"}
	case ColumnTypeLabels:
		return &schema.JSONType{T: "jsonb"}
	default:
		return &schema.StringType{T: "text"}
	}
}

func ReadViewTable(ctx context.Context, columnDef ViewColumnDefList, table string, requestFingerprint string) ([]Row, error) {
	columns := columnDef.QuotedColumns()

	query := ctx.DB().Select(strings.Join(columns, ", ")).Table(table)
	if requestFingerprint != "" {
		query = query.Where("request_fingerprint = ?", requestFingerprint)
	}

	rows, err := query.Rows()
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

			case ColumnTypeLabels:
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

func InsertViewRows(ctx context.Context, table string, columns ViewColumnDefList, rows []Row, requestFingerprint string) error {
	// NOTE: Views refresh frequently; we stage into a temp table and conditionally upsert to avoid
	// unnecessary row churn at the cost of extra complexity.
	if len(rows) == 0 {
		// Delete existing rows for this fingerprint when no new rows are provided
		return ctx.DB().Exec(fmt.Sprintf("DELETE FROM %s WHERE request_fingerprint = ?", pq.QuoteIdentifier(table)), requestFingerprint).Error
	}

	return ctx.DB().Transaction(func(tx *gorm.DB) error {
		tempID := uuid.New()
		tempTable := fmt.Sprintf("tmp_view_rows_%s", strings.ReplaceAll(tempID.String(), "-", ""))

		createTempSQL := fmt.Sprintf(
			"CREATE TEMP TABLE %s (LIKE %s INCLUDING DEFAULTS) ON COMMIT DROP",
			pq.QuoteIdentifier(tempTable),
			pq.QuoteIdentifier(table),
		)
		if err := tx.Exec(createTempSQL).Error; err != nil {
			return fmt.Errorf("failed to create temp table: %w", err)
		}

		quotedColumns := columns.QuotedColumns()
		quotedColumns = append(quotedColumns, pq.QuoteIdentifier("request_fingerprint"))

		paramsPerRow := len(quotedColumns)
		const maxParams = 65535
		batchSize := maxParams / paramsPerRow
		if batchSize < 1 {
			return fmt.Errorf("too many columns (%d) to fit within parameter limit", paramsPerRow)
		}
		if len(rows) > batchSize {
			ctx.Logger.Warnf("InsertViewRows: batching %d rows (batch size %d) to stay under 65,535 parameter limit", len(rows), batchSize)
		}

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
		for start := 0; start < len(rows); start += batchSize {
			end := min(start+batchSize, len(rows))

			insertBuilder := psql.Insert(tempTable).Columns(quotedColumns...)
			for _, row := range rows[start:end] {
				rowWithFingerprint := append(row, requestFingerprint)
				insertBuilder = insertBuilder.Values(rowWithFingerprint...)
			}

			insertSQL, args, err := insertBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build temp insert query: %w", err)
			}
			if err := tx.Exec(insertSQL, args...).Error; err != nil {
				return fmt.Errorf("failed to insert rows into temp table: %w", err)
			}
		}

		pkColumns := columns.PrimaryKey()
		quotedPrimaryKeys := lo.Map(pkColumns, func(col string, _ int) string {
			return pq.QuoteIdentifier(col)
		})

		conflictCols := strings.Join(quotedPrimaryKeys, ", ")
		var updateClauses []string
		var distinctClauses []string
		for _, col := range columns {
			if !col.PrimaryKey {
				quotedCol := pq.QuoteIdentifier(col.Name)
				updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", quotedCol, quotedCol))
				distinctClauses = append(distinctClauses, fmt.Sprintf("%s.%s IS DISTINCT FROM EXCLUDED.%s", pq.QuoteIdentifier(table), quotedCol, quotedCol))
			}
		}

		onConflict := fmt.Sprintf("ON CONFLICT (%s) DO NOTHING", conflictCols)
		if len(updateClauses) > 0 {
			onConflict = fmt.Sprintf(
				"ON CONFLICT (%s) DO UPDATE SET %s WHERE %s",
				conflictCols,
				strings.Join(updateClauses, ", "),
				strings.Join(distinctClauses, " OR "),
			)
		}

		// Move staged rows into the main table using INSERT ... SELECT from the temp table.
		upsertSQL := fmt.Sprintf(
			"INSERT INTO %s (%s) SELECT %s FROM %s %s",
			pq.QuoteIdentifier(table),
			strings.Join(quotedColumns, ", "),
			strings.Join(quotedColumns, ", "),
			pq.QuoteIdentifier(tempTable),
			onConflict,
		)
		if err := tx.Exec(upsertSQL).Error; err != nil {
			return fmt.Errorf("failed to upsert view rows: %w", err)
		}

		var pkEq []string
		for _, pk := range pkColumns {
			q := pq.QuoteIdentifier(pk)
			pkEq = append(pkEq, fmt.Sprintf("t.%s = tmp.%s", q, q))
		}

		deleteSQL := fmt.Sprintf(`
			DELETE FROM %s AS t
			WHERE request_fingerprint = $1
			AND NOT EXISTS (
				SELECT 1 FROM %s AS tmp
				WHERE %s
			)`,
			pq.QuoteIdentifier(table),
			pq.QuoteIdentifier(tempTable),
			strings.Join(pkEq, " AND "),
		)

		if err := tx.Exec(deleteSQL, requestFingerprint).Error; err != nil {
			return fmt.Errorf("failed to delete stale view rows: %w", err)
		}

		return nil
	})
}
