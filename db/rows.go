package db

import (
	"database/sql"
	"fmt"
)

func ScanRows[T ~map[string]any](rows *sql.Rows) ([]T, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	values := make([]any, len(columnNames))
	valuePtrs := make([]any, len(columnNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var result []T
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(T, len(columnNames))
		for i, column := range columnNames {
			row[column] = values[i]
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return result, nil
}
