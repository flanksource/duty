package query

import (
	gocontext "context"
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Query executes a SQL query against the "config_" tables in the database.
func Config(ctx context.Context, sqlQuery string) ([]map[string]any, error) {
	if isValid, err := validateTablesInQuery(sqlQuery, "config_"); err != nil {
		return nil, err
	} else if !isValid {
		return nil, fmt.Errorf("query references restricted tables: %w", err)
	}

	return query(ctx, ctx.Pool(), sqlQuery)
}

// query runs the given SQL query against the provided db connection.
// The rows are returned as a map of columnName=>columnValue.
func query(ctx context.Context, conn *pgxpool.Pool, query string) ([]map[string]any, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(DefaultQueryTimeout)
		defer cancel()
	}

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	columns := rows.FieldDescriptions()
	results := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("error scaning row: %w", err)
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col.Name] = values[i]
		}

		results = append(results, row)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return results, nil
}

func FindConfigIDsByNameNamespaceType(ctx context.Context, namespace, name, configType string) ([]uuid.UUID, error) {
	return lookupIDs(ctx, "config_items", namespace, name, configType)
}
