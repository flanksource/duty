package duty

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/flanksource/commons/logger"
)

// Query runs the given SQL query against the provided db connection.
// The rows are returned as a map of columnName=>columnValue.
func Query(ctx context.Context, conn *sql.DB, query string) ([]map[string]any, error) {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Errorf("failed to close rows: %w", err)
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

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
			row[col] = values[i]
		}

		results = append(results, row)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return results, tx.Commit()
}
