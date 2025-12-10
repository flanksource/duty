package dataquery

import (
	"fmt"
	"time"

	"github.com/flanksource/commons/duration"
	promV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/timberio/go-datemath"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
)

// PrometheusRange defines parameters for running a range query.
type PrometheusRange struct {
	Start string `json:"start" yaml:"start"`
	End   string `json:"end" yaml:"end"`
	Step  string `json:"step" yaml:"step"`
}

func (pr PrometheusRange) toPrometheusRange(now time.Time) (promV1.Range, error) {
	if pr.Start == "" {
		return promV1.Range{}, fmt.Errorf("prometheus range start time is required")
	}
	if pr.End == "" {
		return promV1.Range{}, fmt.Errorf("prometheus range end time is required")
	}
	if pr.Step == "" {
		return promV1.Range{}, fmt.Errorf("prometheus range step is required")
	}

	start, err := datemath.ParseAndEvaluate(pr.Start, datemath.WithNow(now))
	if err != nil {
		return promV1.Range{}, fmt.Errorf("invalid prometheus range start time: %w", err)
	}

	end, err := datemath.ParseAndEvaluate(pr.End, datemath.WithNow(now))
	if err != nil {
		return promV1.Range{}, fmt.Errorf("invalid prometheus range end time: %w", err)
	}

	step, err := duration.ParseDuration(pr.Step)
	if err != nil {
		return promV1.Range{}, fmt.Errorf("invalid prometheus range step: %w", err)
	}

	stepDuration := time.Duration(step)
	if stepDuration <= 0 {
		return promV1.Range{}, fmt.Errorf("prometheus range step must be greater than zero")
	}

	if end.Before(start) {
		return promV1.Range{}, fmt.Errorf("prometheus range end time must be after start time")
	}

	return promV1.Range{
		Start: start,
		End:   end,
		Step:  stepDuration,
	}, nil
}

// +kubebuilder:object:generate=true
// PrometheusQuery defines a Prometheus query configuration
type PrometheusQuery struct {
	connection.PrometheusConnection `json:",inline" yaml:",inline"`

	// Query is the PromQL query string
	Query string `json:"query" yaml:"query"`

	// Range runs a PromQL range query when specified
	Range *PrometheusRange `json:"range,omitempty" yaml:"range,omitempty"`
}

// executePrometheusQuery executes a Prometheus query and returns results
func executePrometheusQuery(ctx context.Context, pq PrometheusQuery) ([]QueryResultRow, error) {
	if pq.Query == "" {
		return nil, fmt.Errorf("prometheus query is required")
	}

	if err := pq.PrometheusConnection.Populate(ctx); err != nil {
		return nil, fmt.Errorf("failed to populate prometheus connection: %w", err)
	}

	client, err := pq.PrometheusConnection.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	result, err := executePromQLQuery(ctx, client, pq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PromQL query: %w", err)
	}

	results, err := transformPrometheusResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to transform prometheus result: %w", err)
	}

	return results, nil
}

// executePromQLQuery executes a PromQL query against Prometheus
func executePromQLQuery(ctx context.Context, client promV1.API, pq PrometheusQuery) (model.Value, error) {
	if pq.Range != nil {
		now := time.Now()
		promRange, err := pq.Range.toPrometheusRange(now)
		if err != nil {
			return nil, err
		}

		result, warnings, err := client.QueryRange(ctx, pq.Query, promRange)
		if err != nil {
			return nil, fmt.Errorf("failed to execute PromQL range query: %w", err)
		}

		if len(warnings) > 0 {
			ctx.Warnf("Prometheus query warnings: %v", warnings)
		}

		return result, nil
	}

	result, warnings, err := client.Query(ctx, pq.Query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to execute PromQL query: %w", err)
	}

	if len(warnings) > 0 {
		ctx.Warnf("Prometheus query warnings: %v", warnings)
	}

	return result, nil
}

// transformPrometheusResult transforms Prometheus model.Value to QueryResultRow format
func transformPrometheusResult(result model.Value) ([]QueryResultRow, error) {
	if result == nil {
		return []QueryResultRow{}, nil
	}

	var results []QueryResultRow

	switch v := result.(type) {
	case model.Vector:
		for _, sample := range v {
			row := QueryResultRow{}

			// Add metric labels
			for label, value := range sample.Metric {
				row[string(label)] = string(value)
			}

			// Add the value
			row["value"] = float64(sample.Value)
			results = append(results, row)
		}

	case model.Matrix:
		for _, sampleStream := range v {
			for _, samplePair := range sampleStream.Values {
				row := QueryResultRow{}

				// Add metric labels
				for label, value := range sampleStream.Metric {
					row[string(label)] = string(value)
				}

				// Add timestamp and value
				row["timestamp"] = samplePair.Timestamp.Time()
				row["value"] = float64(samplePair.Value)
				results = append(results, row)
			}
		}

	case *model.Scalar:
		row := QueryResultRow{
			"value":     float64(v.Value),
			"timestamp": v.Timestamp.Time(),
		}
		results = append(results, row)

	case *model.String:
		row := QueryResultRow{
			"value": v.Value,
		}
		results = append(results, row)

	default:
		return nil, fmt.Errorf("unsupported prometheus result type: %T", result)
	}

	return results, nil
}
