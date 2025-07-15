package dataquery

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
)

// MergeQueryResults merges multiple query result sets into a single result set using the provided SQL query
func MergeQueryResults(ctx context.Context, resultsets []QueryResultSet, mergeQuery string) ([]QueryResultRow, error) {
	if len(resultsets) == 0 {
		return nil, fmt.Errorf("no results to merge")
	}

	if mergeQuery == "" {
		return nil, fmt.Errorf("merge query is required")
	}

	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory SQLite database: %w", err)
	}

	sqlDB, err := sqliteDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	defer sqlDB.Close()

	sqliteCtx := ctx.WithDB(sqliteDB, nil)

	// Create tables for each result set and insert the rows
	for _, resultSet := range resultsets {
		if err := resultSet.CreateDBTable(sqliteCtx); err != nil {
			return nil, fmt.Errorf("failed to create table for result set '%s': %w", resultSet.Name, err)
		}

		if err := resultSet.InsertToDB(sqliteCtx); err != nil {
			return nil, fmt.Errorf("failed to insert data into table '%s': %w", resultSet.Name, err)
		}
	}

	mergedResults, err := mergeResultsets(sqliteCtx, mergeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute merge query: %w", err)
	}

	return mergedResults, nil
}

// mergeResultsets executes the given merge query on an in-memory SQLite database
// containing all the result sets as tables.
func mergeResultsets(ctx context.Context, mergeQuery string) ([]QueryResultRow, error) {
	rows, err := ctx.DB().Raw(mergeQuery).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute merge query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column information: %w", err)
	}

	var results []QueryResultRow
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(QueryResultRow)
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return results, nil
}
