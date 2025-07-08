package query

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
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

func (v *Query) Valid() error {
	queryTypeCount := 0
	if v.Configs != nil && !v.Configs.IsEmpty() {
		queryTypeCount++
	}
	if v.Changes != nil && !v.Changes.IsEmpty() {
		queryTypeCount++
	}
	if v.Prometheus != nil && v.Prometheus.Query != "" {
		queryTypeCount++
	}

	if queryTypeCount == 0 {
		return fmt.Errorf("query has no datasource specified")
	}
	if queryTypeCount > 1 {
		return fmt.Errorf("query has multiple datasources specified, exactly one is required")
	}

	return nil
}

type QueryResultRow map[string]any

// ExecuteQuery executes a single query and returns results with query name
func ExecuteQuery(ctx context.Context, q Query) ([]QueryResultRow, error) {
	if err := q.Valid(); err != nil {
		return nil, err
	}

	var results []QueryResultRow

	if q.Configs != nil && !q.Configs.IsEmpty() {
		configs, err := FindConfigsByResourceSelector(ctx, -1, *q.Configs)
		if err != nil {
			return nil, fmt.Errorf("failed to find configs: %w", err)
		}

		for _, config := range configs {
			results = append(results, config.AsMap())
		}
	} else if q.Changes != nil && !q.Changes.IsEmpty() {
		changes, err := FindConfigChangesByResourceSelector(ctx, -1, *q.Changes)
		if err != nil {
			return nil, fmt.Errorf("failed to find changes: %w", err)
		}

		for _, change := range changes {
			results = append(results, change.AsMap())
		}
	} else if q.Prometheus != nil {
		if q.Prometheus.PrometheusConnection.ConnectionName == "" {
			return nil, fmt.Errorf("prometheus connection name is required")
		}
		if q.Prometheus.Query == "" {
			return nil, fmt.Errorf("prometheus query string is required")
		}

		prometheusResults, err := executePrometheusQuery(ctx, *q.Prometheus)
		if err != nil {
			return nil, fmt.Errorf("failed to execute prometheus query: %w", err)
		}

		results = prometheusResults
	} else {
		return nil, fmt.Errorf("query has not datasource specified")
	}

	return results, nil
}
