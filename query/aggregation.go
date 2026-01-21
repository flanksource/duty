package query

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

// Aggregation function constants
const (
	AggFunctionCount = "COUNT"
	AggFunctionSum   = "SUM"
	AggFunctionAvg   = "AVG"
	AggFunctionMax   = "MAX"
	AggFunctionMin   = "MIN"
)

// AllowedAggregationFunctions lists all permitted aggregation functions
var AllowedAggregationFunctions = []string{
	AggFunctionCount,
	AggFunctionSum,
	AggFunctionAvg,
	AggFunctionMax,
	AggFunctionMin,
}

// NumericAggregationFunctions lists functions that require numeric values
var NumericAggregationFunctions = []string{
	AggFunctionSum,
	AggFunctionAvg,
	AggFunctionMax,
	AggFunctionMin,
}

// Aggregate performs aggregation queries on resources with GROUP BY and aggregation functions
// The select clauses, limits are defined in the query struct.
// The table parameter specifies which table to query (e.g., "config_items", "components", "checks")
func Aggregate(ctx context.Context, table string, query types.AggregatedResourceSelector) ([]types.AggregateRow, error) {
	if err := validateGroupByFields(table, query.GroupBy); err != nil {
		return nil, fmt.Errorf("invalid GROUP BY fields: %w", err)
	}

	if err := validateAggregationFields(table, query.Aggregates); err != nil {
		return nil, fmt.Errorf("invalid aggregation fields: %w", err)
	}

	db := ctx.DB().Table(table)

	if !query.ResourceSelector.IsEmpty() {
		var err error
		db, err = SetResourceSelectorClause(ctx, query.ResourceSelector.Canonical(), db, table)
		if err != nil {
			return nil, fmt.Errorf("failed to apply resource selector: %w", err)
		}
	}

	selectClause := BuildSelectClause(query.GroupBy, query.Aggregates)
	db = db.Select(selectClause)

	if len(query.GroupBy) > 0 {
		groupByClause := buildGroupByClause(query.GroupBy)
		db = db.Clauses(groupByClause)
	}

	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	db = db.Limit(limit)

	var results []types.AggregateRow
	rows, err := db.Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result := make(types.AggregateRow)
		for i, col := range columns {
			result[col] = values[i]
		}
		results = append(results, result)
	}

	return results, nil
}

// BuildSelectClause constructs the SELECT clause with GROUP BY fields and aggregations
func BuildSelectClause(groupBy []string, aggregates []types.AggregationField) string {
	var parts []string

	for _, field := range groupBy {
		if strings.Contains(field, ".") {
			selector, alias := BuildJSONFieldSelector(field)
			if alias != "" {
				parts = append(parts, fmt.Sprintf(`%s as "%s"`, selector, alias))
			} else {
				parts = append(parts, selector)
			}
		} else {
			parts = append(parts, field)
		}
	}

	for _, agg := range aggregates {
		var aggClause string
		if strings.ToUpper(agg.Function) == AggFunctionCount && agg.Field == "*" {
			aggClause = fmt.Sprintf("COUNT(*) AS %s", agg.Alias)
		} else if strings.Contains(agg.Field, ".") {
			selector, _ := BuildJSONFieldSelector(agg.Field)
			if isNumericAggregation(agg.Function) {
				aggClause = fmt.Sprintf("%s(CAST(%s AS NUMERIC)) AS %s", strings.ToUpper(agg.Function), selector, agg.Alias)
			} else {
				aggClause = fmt.Sprintf("%s(%s) AS %s", strings.ToUpper(agg.Function), selector, agg.Alias)
			}
		} else {
			aggClause = fmt.Sprintf("%s(%s) AS %s", strings.ToUpper(agg.Function), agg.Field, agg.Alias)
		}

		parts = append(parts, aggClause)
	}

	return strings.Join(parts, ", ")
}

// BuildJSONFieldSelector creates SQL for accessing JSON fields and returns both the selector and alias
// This assumes the field has already been validated by isValidFieldForQuery
func BuildJSONFieldSelector(field string) (selector, alias string) {
	parts := strings.SplitN(field, ".", 2)
	if len(parts) != 2 {
		return field, ""
	}

	column := parts[0]
	path := parts[1]
	alias = strings.ReplaceAll(path, ".", "_")

	// Special case for properties which uses jsonb_path_query_first
	if column == "properties" {
		selector = fmt.Sprintf("jsonb_path_query_first(properties, '$.%s')", path)
		return selector, alias
	}

	// Generic handling for all JSON map columns
	// Handle nested paths like config.author.name -> config->'author'->>'name'
	pathParts := strings.Split(path, ".")
	if len(pathParts) == 1 {
		selector = fmt.Sprintf(`%s->>'%s'`, column, path)
		alias = path
	} else {
		// Build nested JSON path
		var jsonPath strings.Builder
		jsonPath.WriteString(column)
		for i, part := range pathParts {
			if i == len(pathParts)-1 {
				jsonPath.WriteString(fmt.Sprintf(`->>'%s'`, part))
			} else {
				jsonPath.WriteString(fmt.Sprintf(`->'%s'`, part))
			}
		}
		selector = jsonPath.String()
	}

	return selector, alias
}

// buildGroupByClause constructs the GROUP BY clause
func buildGroupByClause(groupBy []string) clause.Expression {
	var groupByClause clause.GroupBy
	for i, field := range groupBy {
		if strings.Contains(field, ".") {
			groupByClause.Columns = append(groupByClause.Columns,
				clause.Column{
					Name: strconv.Itoa(i + 1),
					Raw:  true,
				},
			)
		} else {
			groupByClause.Columns = append(groupByClause.Columns, clause.Column{Name: field})
		}
	}

	return groupByClause
}

// isNumericAggregation checks if the aggregation function requires numeric values
func isNumericAggregation(function string) bool {
	function = strings.ToUpper(function)
	return slices.Contains(NumericAggregationFunctions, function)
}

// isValidFieldForQuery checks if a field is in the query model's allowed columns or JSON fields
func isValidFieldForQuery(qm QueryModel, field string) bool {
	// Check regular columns
	if slices.Contains(qm.Columns, field) {
		return true
	}

	// For fields with dots, check if they're valid JSON map column accesses
	if strings.Contains(field, ".") {
		parts := strings.SplitN(field, ".", 2)
		if len(parts) == 2 {
			column := parts[0]

			// Check JSON map columns
			if slices.Contains(qm.JSONMapColumns, column) {
				return true
			}

			// Check properties if supported
			if qm.HasProperties && column == "properties" {
				return true
			}
		}
	}

	return false
}

// validateAggregationFields validates all aggregation fields for security and correctness
func validateAggregationFields(table string, aggregates []types.AggregationField) error {
	qm, err := GetModelFromTable(table)
	if err != nil {
		return fmt.Errorf("unsupported table %s: %w", table, err)
	}

	for _, agg := range aggregates {
		if err := validateAggregationField(qm, agg); err != nil {
			return err
		}
	}
	return nil
}

// validateAggregationField validates a single aggregation field
func validateAggregationField(qm QueryModel, agg types.AggregationField) error {
	// Validate function name
	if !isValidAggregationFunction(agg.Function) {
		return fmt.Errorf("invalid aggregation function: %s", agg.Function)
	}

	// Validate alias
	if agg.Alias == "" {
		return fmt.Errorf("aggregation alias is required")
	}

	// Special case for COUNT(*)
	if strings.ToUpper(agg.Function) == AggFunctionCount && agg.Field == "*" {
		return nil
	}

	// Validate field access
	if !isValidFieldForQuery(qm, agg.Field) {
		return fmt.Errorf("aggregation field '%s' is not allowed", agg.Field)
	}

	return nil
}

// validateGroupByFields validates all GROUP BY fields for security and correctness
func validateGroupByFields(table string, fields []string) error {
	qm, err := GetModelFromTable(table)
	if err != nil {
		return fmt.Errorf("unsupported table %s: %w", table, err)
	}

	for _, field := range fields {
		if !isValidFieldForQuery(qm, field) {
			return fmt.Errorf("GROUP BY field '%s' is not allowed for table '%s'", field, table)
		}
	}
	return nil
}

// isValidAggregationFunction checks if the function is in the allowed list
func isValidAggregationFunction(function string) bool {
	function = strings.ToUpper(function)
	return slices.Contains(AllowedAggregationFunctions, function)
}
