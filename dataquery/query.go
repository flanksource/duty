package dataquery

import (
	"fmt"

	"github.com/flanksource/duty/context"
)

// +kubebuilder:object:generate=true
type Query struct {
	// Prometheus queries metrics from Prometheus
	Prometheus *PrometheusQuery `json:"prometheus,omitempty" yaml:"prometheus,omitempty"`

	// SQL runs arbitrary SQL queries against a configured SQL connection
	SQL *SQLQuery `json:"sql,omitempty" yaml:"sql,omitempty"`
}

func (v *Query) IsEmpty() bool {
	return v.Prometheus == nil && v.SQL == nil
}

type QueryResultRow map[string]any

// ExecuteQuery executes a single query and returns results with query name
func ExecuteQuery(ctx context.Context, q Query) ([]QueryResultRow, error) {
	var results []QueryResultRow
	switch {
	case q.Prometheus != nil && q.SQL != nil:
		return nil, fmt.Errorf("multiple data sources specified")
	case q.Prometheus != nil:
		prometheusResults, err := executePrometheusQuery(ctx, *q.Prometheus)
		if err != nil {
			return nil, fmt.Errorf("failed to execute prometheus query: %w", err)
		}

		results = prometheusResults
	case q.SQL != nil:
		sqlResults, err := executeSQLQuery(ctx, *q.SQL)
		if err != nil {
			return nil, fmt.Errorf("failed to execute sql query: %w", err)
		}

		results = sqlResults
	default:
		return nil, fmt.Errorf("query has no data source specified")
	}

	return results, nil
}

// RunSQL runs a query and returns the results
func RunSQL(ctx context.Context, query string) ([]QueryResultRow, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	rows, err := ctx.DB().Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
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
