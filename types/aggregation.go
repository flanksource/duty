package types

// AggregationField defines a single aggregation operation
type AggregationField struct {
	Function string `json:"function"` // COUNT, SUM, AVG, MAX, MIN
	Field    string `json:"field"`    // Column name or "*" for COUNT(*)
	Alias    string `json:"alias"`    // Resulting field name in output
}

// +kubebuilder:object:generate=true
// AggregatedResourceSelector combines filtering and aggregation requirements
type AggregatedResourceSelector struct {
	ResourceSelector `json:",inline"` // WHERE clause (reuses existing logic)

	// +kubebuilder:validation:Optional
	GroupBy []string `json:"groupBy,omitempty"` // GROUP BY fields

	// +kubebuilder:validation:Optional
	Aggregates []AggregationField `json:"aggregates,omitempty"` // SELECT aggregations
}

// AggregateRow represents a single row in the aggregation result
type AggregateRow map[string]any
