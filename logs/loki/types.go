package loki

import (
	"net/url"
	"strconv"
	"time"

	"github.com/flanksource/commons/logger"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/timberio/go-datemath"

	"github.com/flanksource/duty/logs"
)

// Response represents the top-level response from Loki's query_range API.
type Response struct {
	Status    string       `json:"status"`
	Data      Data         `json:"data"`
	ErrorType v1.ErrorType `json:"errorType,omitempty"`
	Error     string       `json:"error,omitempty"`
}

func (t *Response) ToLogResult(mappingConfig logs.FieldMappingConfig) logs.LogResult {
	output := logs.LogResult{
		Metadata: t.Data.Stats,
	}

	for _, result := range t.Data.Result {
		for _, v := range result.Values {
			if len(v) != 2 {
				continue
			}

			firstObserved, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				logger.Errorf("loki:failed to parse first observed %s: %v", v[0], err)
				continue
			}

			line := &logs.LogLine{
				Count:         1,
				FirstObserved: time.Unix(0, firstObserved),
				Message:       v[1],
				Labels:        result.Stream,
			}

			for k, v := range result.Stream {
				if err := logs.MapFieldToLogLine(k, v, line, mappingConfig); err != nil {
					// Log or handle mapping error? For now, just log it.
					logger.Errorf("Error mapping field %s for log %s: %v", k, line.ID, err)
				}
			}

			line.SetHash()
			output.Logs = append(output.Logs, line)
		}
	}

	return output
}

type Result struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// Data holds the actual query results and statistics.
type Data struct {
	ResultType string         `json:"resultType"`
	Stats      map[string]any `json:"stats"`

	// Logs per label (aka. stream)
	Result []Result `json:"result"`
}

// Request represents available parameters for Loki queries.
//
// +kubebuilder:object:generate=true
type Request struct {
	logs.LogsRequestBase `json:",inline" template:"true"`

	// Query is the LogQL query to perform
	Query string `json:"query,omitempty" template:"true"`

	// Since is a duration used to calculate start relative to end.
	// If end is in the future, start is calculated as this duration before now.
	// Any value specified for start supersedes this parameter.
	Since string `json:"since,omitempty"`

	// Step is the Query resolution step width in duration format or float number of seconds
	Step string `json:"step,omitempty"`

	// Only return entries at (or greater than) the specified interval, can be a duration format or float number of seconds
	Interval string `json:"interval,omitempty"`

	// Direction is the direction of the query. "forward" or "backward" (default)
	Direction string `json:"direction,omitempty"`
}

// Params returns the URL query parameters for the Loki request
func (r *Request) Params() url.Values {
	// https://grafana.com/docs/loki/latest/reference/loki-http-api/#query-logs-within-a-range-of-time
	params := url.Values{}

	if r.Query != "" {
		params.Set("query", r.Query)
	}
	if r.Limit != "" {
		params.Set("limit", r.Limit)
	}
	if s, err := r.GetStart(); err == nil {
		params.Set("start", s.Format(time.RFC3339))
	}
	if e, err := r.GetEnd(); err == nil {
		params.Set("end", e.Format(time.RFC3339))
	}
	if r.Since != "" {
		params.Set("since", r.Since)
	}
	if r.Step != "" {
		params.Set("step", r.Step)
	}
	if r.Interval != "" {
		params.Set("interval", r.Interval)
	}
	if r.Direction != "" {
		params.Set("direction", r.Direction)
	}

	return params
}

// StreamRequest represents parameters for Loki streaming queries via tail endpoint.
//
// +kubebuilder:object:generate=true
type StreamRequest struct {
	// Query is the LogQL query to perform
	Query string `json:"query,omitempty" template:"true"`

	// DelayFor is the number of seconds to delay retrieving logs (default 0, max 5)
	DelayFor int `json:"delayFor,omitempty"`

	// Limit is the maximum number of entries to return per stream in the initial response when connecting (default 100).
	// This only affects historical entries sent immediately upon connection, not the ongoing stream of new entries.
	Limit int `json:"limit,omitempty"`

	// Start is the start time for the query (default one hour ago)
	// Supports Datemath
	Start string `json:"start,omitempty"`
}

// Params returns the URL query parameters for the Loki streaming request
func (r *StreamRequest) Params() url.Values {
	// https://grafana.com/docs/loki/latest/reference/loki-http-api/#stream-logs
	params := url.Values{}

	if r.Query != "" {
		params.Set("query", r.Query)
	}
	if r.DelayFor > 0 {
		params.Set("delay_for", strconv.Itoa(r.DelayFor))
	}
	if r.Limit > 0 {
		params.Set("limit", strconv.Itoa(r.Limit))
	}
	if r.Start != "" {
		if s, err := r.GetStart(); err == nil {
			params.Set("start", s.Format(time.RFC3339))
		}
	}

	return params
}

// GetStart parses the start time using datemath
func (r *StreamRequest) GetStart() (time.Time, error) {
	if r.Start == "" {
		return time.Now().Add(-1 * time.Hour), nil
	}
	return datemath.ParseAndEvaluate(r.Start, datemath.WithNow(time.Now()))
}

// StreamResponse represents the response from Loki's tail endpoint
type StreamResponse struct {
	Streams        []Result       `json:"streams"`
	DroppedEntries []DroppedEntry `json:"dropped_entries,omitempty"`
}

// DroppedEntry represents entries that were not included in the stream
type DroppedEntry struct {
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

type StreamItem struct {
	LogLine *logs.LogLine
	Error   error
}
