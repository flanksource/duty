package dataquery

import (
	"fmt"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/db"
)

// +kubebuilder:object:generate=true
// SQLQuery defines a SQL query configuration
type SQLQuery struct {
	connection.SQLConnection `json:",inline" yaml:",inline"`

	// Query is the SQL query string to execute against the configured connection.
	Query string `json:"query" yaml:"query"`
}

// executeSQLQuery executes a SQL query using the configured connection and returns the results.
func executeSQLQuery(ctx context.Context, sqlQuery SQLQuery) ([]QueryResultRow, error) {
	if sqlQuery.Query == "" {
		return nil, fmt.Errorf("sql query is required")
	}

	if err := sqlQuery.HydrateConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to hydrate sql connection: %w", err)
	}

	client, err := sqlQuery.Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create sql client: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			ctx.Warnf("failed to close sql connection: %v", err)
		}
	}()

	rows, err := client.QueryContext(ctx, sqlQuery.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute sql query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			ctx.Warnf("failed to close sql rows: %v", err)
		}
	}()

	scannedRows, err := db.ScanRows[QueryResultRow](rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan sql rows: %w", err)
	}

	return scannedRows, nil
}
