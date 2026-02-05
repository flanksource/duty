package bigquery

import (
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/flanksource/commons/utils"

	"github.com/flanksource/duty/logs"
)

// Request represents parameters for BigQuery log queries.
//
// +kubebuilder:object:generate=true
type Request struct {
	// Query is the raw SQL query to execute against the BigQuery table.
	Query string `json:"query,omitempty" template:"true"`
}

// Response represents the response from BigQuery for log data.
type Response struct {
	// Rows contains the actual log data.
	Rows []Row `json:"rows,omitempty"`

	// Schema contains the column names in the same order as the row values.
	Schema bigquery.Schema `json:"schema,omitempty"`

	// NextPageToken for pagination.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// TotalRows is the total number of rows in the result (if available).
	TotalRows int64 `json:"totalRows,omitempty"`

	// JobStats contains query execution statistics.
	JobStats *JobStats `json:"jobStats,omitempty"`
}

// JobStats contains BigQuery job execution statistics.
type JobStats struct {
	// BytesProcessed is the total bytes processed by the query.
	BytesProcessed int64 `json:"bytesProcessed,omitempty"`

	// BytesBilled is the total bytes billed for the query.
	BytesBilled int64 `json:"bytesBilled,omitempty"`

	// CreationTime is when the job was created.
	CreationTime time.Time `json:"creationTime,omitempty"`

	// StartTime is when the job execution started.
	StartTime time.Time `json:"startTime,omitempty"`

	// EndTime is when the job execution ended.
	EndTime time.Time `json:"endTime,omitempty"`

	// TotalSlotMs is the total slot milliseconds consumed.
	TotalSlotMs int64 `json:"totalSlotMs,omitempty"`
}

// ToLogResult converts the BigQuery response to the standard LogResult format.
func (r *Response) ToLogResult(mappingConfig logs.FieldMappingConfig) (logs.LogResult, error) {
	output := logs.LogResult{
		Metadata: make(map[string]any),
	}

	// Add job statistics to metadata
	if r.JobStats != nil {
		output.Metadata["bytesProcessed"] = r.JobStats.BytesProcessed
		output.Metadata["bytesBilled"] = r.JobStats.BytesBilled
		output.Metadata["totalSlotMs"] = r.JobStats.TotalSlotMs
		output.Metadata["executionTime"] = r.JobStats.EndTime.Sub(r.JobStats.StartTime).String()
	}
	output.Metadata["totalRows"] = r.TotalRows
	output.Metadata["nextPageToken"] = r.NextPageToken

	for _, row := range r.Rows {
		line := &logs.LogLine{
			Count: 1,
		}

		for i, value := range row {
			if i >= len(r.Schema) {
				continue
			}

			fieldSchema := r.Schema[i]
			if value != nil {
				if err := logs.MapFieldToLogLine(fieldSchema.Name, value, line, mappingConfig); err != nil {
					return output, fmt.Errorf("failed to map field %s: %w", fieldSchema.Name, err)
				}
			}
		}

		// Use the entire row as message fallback if no message was mapped
		if line.Message == "" {
			if m, err := utils.Stringify(row); err == nil {
				line.Message = m
			}
		}

		// Set hash for deduplication
		line.SetHash()
		output.Logs = append(output.Logs, line)
	}

	return output, nil
}

type Row []bigquery.Value
