package dataquery

import (
	"fmt"

	"github.com/flanksource/duty/context"
)

// +kubebuilder:object:generate=true
type Query struct {
	// Prometheus queries metrics from Prometheus
	Prometheus *PrometheusQuery `json:"prometheus,omitempty" yaml:"prometheus,omitempty"`
}

func (v *Query) IsEmpty() bool {
	return v.Prometheus == nil
}

type QueryResultRow map[string]any

// ExecuteQuery executes a single query and returns results with query name
func ExecuteQuery(ctx context.Context, q Query) ([]QueryResultRow, error) {
	var results []QueryResultRow
	if q.Prometheus != nil {
		prometheusResults, err := executePrometheusQuery(ctx, *q.Prometheus)
		if err != nil {
			return nil, fmt.Errorf("failed to execute prometheus query: %w", err)
		}

		results = prometheusResults
	} else {
		return nil, fmt.Errorf("query has no data source specified")
	}

	return results, nil
}
