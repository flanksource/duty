package query

import (
	gocontext "context"
	"fmt"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xwb1989/sqlparser"
)

// Recursively inspects a given SQLNode and applies the supplied inspector function to each node.
func inspect(node sqlparser.SQLNode, inspector func(node sqlparser.TableName) bool) bool {
	switch node := node.(type) {
	case *sqlparser.Select:
		for _, expr := range node.From {
			if !inspect(expr, inspector) {
				return false
			}
		}

		if node.Where != nil {
			return inspect(node.Where, inspector)
		}

	case *sqlparser.AliasedTableExpr:
		return inspect(node.Expr, inspector)

	case *sqlparser.Where:
		return inspect(node.Expr, inspector)

	case sqlparser.TableName:
		return inspector(node)

	case *sqlparser.JoinTableExpr:
		if !inspect(node.LeftExpr, inspector) {
			return false
		}
		return inspect(node.RightExpr, inspector)

	case *sqlparser.Union:
		if !inspect(node.Left, inspector) {
			return false
		}
		return inspect(node.Right, inspector)

	case *sqlparser.ComparisonExpr:
		if !inspect(node.Left, inspector) {
			return false
		}
		return inspect(node.Right, inspector)

	case *sqlparser.Subquery:
		return inspect(node.Select, inspector)

	case *sqlparser.ColName, *sqlparser.SQLVal:
		// Do nothing
		return true

	default:
		logger.Debugf("unexpected node of type: %T", node)
	}

	return false
}

// validateTablesInQuery checks if a SQL query only uses tables whose names are prefixed by
// prefixes in the allowedPrefix parameter.
//
// It currently only supports SELECT queries.
func validateTablesInQuery(query string, allowedPrefix ...string) (bool, error) {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return false, fmt.Errorf("failed to parse SQL query: %w", err)
	}

	var isValid bool
	inspect(stmt, func(node sqlparser.TableName) bool {
		for _, prefix := range allowedPrefix {
			if strings.HasPrefix(node.Name.String(), prefix) {
				isValid = true
				return true // Continue traversing. Need to verify all the referenced tables.
			}
		}

		isValid = false
		return false // Stop traversing
	})

	return isValid, nil
}

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
