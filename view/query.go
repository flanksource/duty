package view

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
// Query defines a query configuration for populating the view
type Query struct {
	// Configs queries config items
	Configs *types.ResourceSelector `json:"configs,omitempty" yaml:"configs,omitempty"`

	// Changes queries config changes
	Changes *types.ResourceSelector `json:"changes,omitempty" yaml:"changes,omitempty"`

	// Prometheus queries metrics from Prometheus
	Prometheus *PrometheusQuery `json:"prometheus,omitempty" yaml:"prometheus,omitempty"`
}

func (v *Query) IsEmpty() bool {
	return v.Configs == nil && v.Changes == nil && v.Prometheus == nil
}

// QueryResult represents all results from a single query
type QueryResult struct {
	Name string
	Rows []QueryResultRow
}

type QueryResultRow map[string]any

// executeQuery executes a single query and returns results with query name
func ExecuteQuery(ctx context.Context, q Query) ([]QueryResultRow, error) {
	var results []QueryResultRow

	if q.Configs != nil && !q.Configs.IsEmpty() {
		configs, err := query.FindConfigsByResourceSelector(ctx, -1, *q.Configs)
		if err != nil {
			return nil, fmt.Errorf("failed to find configs: %w", err)
		}

		for _, config := range configs {
			results = append(results, config.AsMap())
		}
	} else if q.Changes != nil && !q.Changes.IsEmpty() {
		changes, err := query.FindConfigChangesByResourceSelector(ctx, -1, *q.Changes)
		if err != nil {
			return nil, fmt.Errorf("failed to find changes: %w", err)
		}

		for _, change := range changes {
			results = append(results, change.AsMap())
		}
	} else if q.Prometheus != nil {
		prometheusResults, err := executePrometheusQuery(ctx, *q.Prometheus)
		if err != nil {
			return nil, fmt.Errorf("failed to execute prometheus query: %w", err)
		}

		results = prometheusResults
	} else {
		return nil, fmt.Errorf("view query has not datasource specified")
	}

	return results, nil
}
