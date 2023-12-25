package query

import (
	"fmt"
	"strings"

	"github.com/flanksource/commons/logger"
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
