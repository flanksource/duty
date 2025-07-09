package dataquery

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/flanksource/commons/collections"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
)

// MergeOperation represents the type of merge operation
type MergeOperation string

const (
	MergeLeftJoin MergeOperation = "LEFT_JOIN"
	MergeUnion    MergeOperation = "UNION"
)

// JoinCondition represents a join condition between two tables
type JoinCondition struct {
	LeftTable   string `json:"left_table" yaml:"left_table"`
	LeftColumn  string `json:"left_column" yaml:"left_column"`
	RightTable  string `json:"right_table" yaml:"right_table"`
	RightColumn string `json:"right_column" yaml:"right_column"`
}

// MergeSpec defines how to merge query result sets
type MergeSpec struct {
	Operation MergeOperation `json:"operation" yaml:"operation"`

	// For JOIN operations
	JoinConditions []JoinCondition `json:"join_conditions,omitempty" yaml:"join_conditions,omitempty"`
}

// MergeQueryResults merges multiple query result sets into a single result set using the specified merge strategy
func MergeQueryResults(ctx context.Context, resultsets []QueryResultSet, merge MergeSpec) ([]QueryResultRow, error) {
	if len(resultsets) == 0 {
		return nil, fmt.Errorf("no results to merge")
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

	mergedResults, err := mergeResultsets(sqliteCtx, resultsets, merge)
	if err != nil {
		return nil, fmt.Errorf("failed to execute merge query: %w", err)
	}

	return mergedResults, nil
}

// mergeResultsets builds and executes the SQL query to merge tables
func mergeResultsets(ctx context.Context, resultsets []QueryResultSet, merge MergeSpec) ([]QueryResultRow, error) {
	var query string
	var err error

	switch merge.Operation {
	case MergeLeftJoin:
		query, err = buildJoinQuery(resultsets, merge)
	case MergeUnion:
		query, err = buildUnionQuery(resultsets)
	default:
		return nil, fmt.Errorf("unsupported merge operation: %s", merge.Operation)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build merge query: %w", err)
	}

	rows, err := ctx.DB().Raw(query).Rows()
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
			if values[i] != nil {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return results, nil
}

// buildJoinQuery constructs a JOIN query using Squirrel
func buildJoinQuery(resultsets []QueryResultSet, merge MergeSpec) (string, error) {
	if len(resultsets) == 0 {
		return "", fmt.Errorf("no result sets provided for join")
	}

	query := squirrel.Select().From(resultsets[0].Name)

	// Alias column names with table prefix
	for _, rs := range resultsets {
		if len(rs.Results) > 0 {
			for col := range rs.Results[0] {
				aliasName := fmt.Sprintf("%s.%s", rs.Name, col)
				query = query.Column(fmt.Sprintf(`"%s"."%s" AS "%s"`, rs.Name, col, aliasName))
			}
		}
	}

	for _, joinCond := range merge.JoinConditions {
		joinClause := fmt.Sprintf(`"%s" ON "%s"."%s" = "%s"."%s"`,
			joinCond.RightTable,
			joinCond.LeftTable, joinCond.LeftColumn,
			joinCond.RightTable, joinCond.RightColumn)
		query = query.LeftJoin(joinClause)
	}

	sql, _, err := query.ToSql()
	return sql, err
}

// buildUnionQuery constructs a UNION query using Squirrel
func buildUnionQuery(resultsets []QueryResultSet) (string, error) {
	if len(resultsets) == 0 {
		return "", fmt.Errorf("no result sets provided for union")
	}

	var queries []string

	for _, rs := range resultsets {
		query := squirrel.Select().From(rs.Name)

		if len(rs.Results) > 0 {
			columns := collections.MapKeys(rs.Results[0])
			sort.Strings(columns) // must sort so we get the same column order on UNION
			query = query.Columns(columns...)
		}

		sql, _, err := query.ToSql()
		if err != nil {
			return "", fmt.Errorf("failed to build query for table %s: %w", rs.Name, err)
		}

		queries = append(queries, sql)
	}

	return strings.Join(queries, " UNION "), nil
}
