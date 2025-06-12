package gcpcloudlogging

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/flanksource/commons/collections"
	"github.com/timberio/go-datemath"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	loggingapi "google.golang.org/api/logging/v2"
	"google.golang.org/api/option"

	"github.com/flanksource/duty/connection"
	dutycxt "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
)

// Request represents available parameters for GCP Cloud Logging queries.
//
// +kubebuilder:object:generate=true
type Request struct {
	logs.LogsRequestBase `json:",inline" template:"true"`

	// Filter is the filter to perform
	Filter string `json:"filter,omitempty" template:"true"`
}

// StreamRequest represents parameters for GCP Cloud Logging streaming queries via entries:tail endpoint.
//
// +kubebuilder:object:generate=true
type StreamRequest struct {
	// Filter is the filter expression using GCP Cloud Logging filter syntax
	Filter string `json:"filter,omitempty" template:"true"`

	// BufferWindow is the buffer window in seconds for streaming (default 2)
	BufferWindow int `json:"bufferWindow,omitempty"`

	// Start is the start time for the query (default now)
	// Supports Datemath
	Start string `json:"start,omitempty"`
}

// GetStart parses the start time using datemath
func (r *StreamRequest) GetStart() (time.Time, error) {
	if r.Start == "" {
		return time.Now(), nil
	}
	return datemath.ParseAndEvaluate(r.Start, datemath.WithNow(time.Now()))
}

// StreamItem represents a single item from the streaming response
type StreamItem struct {
	LogLine *logs.LogLine
	Error   error
}

type cloudLogging struct {
	client        *logadmin.Client
	loggingClient *loggingapi.Service
	mappingConfig *logs.FieldMappingConfig
	connection    connection.GCPConnection
}

func (gcp *cloudLogging) Close() error {
	return gcp.client.Close()
}

func New(ctx dutycxt.Context, conn connection.GCPConnection, mappingConfig *logs.FieldMappingConfig) (*cloudLogging, error) {
	var opts []option.ClientOption
	if err := conn.HydrateConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to hydrate connection: %w", err)
	}

	if conn.Credentials != nil && !conn.Credentials.IsEmpty() {
		c, err := google.CredentialsFromJSON(ctx, []byte(conn.Credentials.ValueStatic))
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		opts = append(opts, option.WithCredentials(c))
	}

	adminClient, err := logadmin.NewClient(ctx, conn.Project, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging admin client: %w", err)
	}

	loggingClient, err := loggingapi.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging service client: %w", err)
	}

	return &cloudLogging{
		client:        adminClient,
		loggingClient: loggingClient,
		mappingConfig: mappingConfig,
		connection:    conn,
	}, nil
}

func (gcp *cloudLogging) Search(ctx dutycxt.Context, request Request) (*logs.LogResult, error) {
	var maxLogLines int = 1000
	if request.Limit != "" {
		if l, err := strconv.ParseInt(request.Limit, 10, 32); err == nil && l > 0 {
			maxLogLines = int(l)
		}
	}

	mappingConfig := defaultFieldMappingConfig
	if gcp.mappingConfig != nil {
		mappingConfig = gcp.mappingConfig.WithDefaults(defaultFieldMappingConfig)
	}

	result := &logs.LogResult{
		Logs: make([]*logs.LogLine, 0),
	}

	filterParts := []string{}
	if request.Filter != "" {
		filterParts = append(filterParts, request.Filter)
	}

	if request.Start != "" {
		startTime, err := request.GetStart()
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		filterParts = append(filterParts, fmt.Sprintf(`timestamp >= "%s"`, startTime.Format("2006-01-02T15:04:05Z")))
	}

	if request.End != "" {
		endTime, err := request.GetEnd()
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}
		filterParts = append(filterParts, fmt.Sprintf(`timestamp <= "%s"`, endTime.Format("2006-01-02T15:04:05Z")))
	}

	var combinedFilter string
	if len(filterParts) > 0 {
		wrappedParts := make([]string, len(filterParts))
		for i, part := range filterParts {
			wrappedParts[i] = fmt.Sprintf("(%s)", part)
		}
		combinedFilter = strings.Join(wrappedParts, " AND ")
	}

	pageSize := min(maxLogLines, 1000)
	opts := []logadmin.EntriesOption{
		logadmin.PageSize(int32(pageSize)),
		logadmin.NewestFirst(),
	}

	if combinedFilter != "" {
		opts = append(opts, logadmin.Filter(combinedFilter))
	}

	it := gcp.client.Entries(ctx, opts...)
	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to list access log entries: %w", err)
		}

		if entry.Payload == nil {
			continue
		}

		line := &logs.LogLine{
			Count: 1,
		}

		if entry.InsertID != "" {
			line.ID = entry.InsertID
		}
		if !entry.Timestamp.IsZero() {
			line.FirstObserved = entry.Timestamp
		}
		if entry.Severity != 0 {
			line.Severity = entry.Severity.String()
		}
		if entry.LogName != "" {
			line.Source = entry.LogName
		}

		switch payload := entry.Payload.(type) {
		case string:
			line.Message = payload
		default:
			payloadMap, err := collections.ToJSONMap(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to convert payload to JSON map: %w", err)
			}

			for k, v := range payloadMap {
				if err := logs.MapFieldToLogLine(k, v, line, mappingConfig); err != nil {
					ctx.Warnf("Error mapping field %s for log %s: %v", k, line.ID, err)
				}
			}
		}

		if entry.Resource != nil && entry.Resource.Labels != nil {
			for k, v := range entry.Resource.Labels {
				resourceKey := "resource." + k
				if err := logs.MapFieldToLogLine(resourceKey, v, line, mappingConfig); err != nil {
					ctx.Warnf("Error mapping resource field %s for log %s: %v", resourceKey, line.ID, err)
				}
			}
		}

		if entry.Labels != nil {
			for k, v := range entry.Labels {
				labelKey := "label." + k
				if err := logs.MapFieldToLogLine(labelKey, v, line, mappingConfig); err != nil {
					ctx.Warnf("Error mapping label field %s for log %s: %v", labelKey, line.ID, err)
				}
			}
		}

		line.SetHash()
		result.Logs = append(result.Logs, line)
		if len(result.Logs) >= maxLogLines {
			break
		}
	}

	return result, nil
}

func (gcp *cloudLogging) Stream(ctx context.Context, request StreamRequest) (<-chan StreamItem, error) {
	mappingConfig := defaultFieldMappingConfig
	if gcp.mappingConfig != nil {
		mappingConfig = gcp.mappingConfig.WithDefaults(defaultFieldMappingConfig)
	}

	// Prepare the tail request
	var resourceNames []string
	if gcp.connection.Project != "" {
		resourceNames = append(resourceNames, fmt.Sprintf("projects/%s", gcp.connection.Project))
	}

	req := &loggingapi.TailLogEntriesRequest{
		ResourceNames: resourceNames,
	}

	if request.Filter != "" {
		req.Filter = request.Filter
	}

	if request.BufferWindow > 0 {
		req.BufferWindow = fmt.Sprintf("%d", request.BufferWindow*1000)
	} else {
		req.BufferWindow = "2000"
	}

	itemChan := make(chan StreamItem)

	go func() {
		defer close(itemChan)

		// Use periodic polling since the Go client doesn't support true streaming for tail
		ticker := time.NewTicker(time.Duration(max(request.BufferWindow, 1)) * time.Millisecond)
		if request.BufferWindow <= 0 {
			ticker = time.NewTicker(2 * time.Second)
		}
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Make the tail call
				tailCall := gcp.loggingClient.Entries.Tail(req)
				tailCall.Context(ctx)

				// https://stackoverflow.com/questions/76765079/v2-entriestail-always-returns-invalid-argument
				// https: //cloud.google.com/logging/docs/reference/v2/rest/v2/entries/tail?apix=true&apix_params=%7B%22resource%22%3A%5B%7B%22resourceNames%22%3A%5B%22projects%2Fworkload-prod-eu-02%22%5D%7D%5D%7D#Reason
				resp, err := tailCall.Do()
				if err != nil {
					select {
					case itemChan <- StreamItem{Error: fmt.Errorf("tail request failed: %w", err)}:
					case <-ctx.Done():
						return
					}
					continue
				}

				// Process each log entry in the response
				for _, entry := range resp.Entries {
					logLine := gcp.convertAPILogEntryToLogLine(entry, mappingConfig)
					if logLine != nil {
						select {
						case itemChan <- StreamItem{LogLine: logLine}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return itemChan, nil
}

// convertLogEntryToLogLine converts a logging.Entry to a LogLine
func (gcp *cloudLogging) convertLogEntryToLogLine(entry *logging.Entry, mappingConfig logs.FieldMappingConfig) *logs.LogLine {
	if entry.Payload == nil {
		return nil
	}

	line := &logs.LogLine{
		Count: 1,
	}

	if entry.InsertID != "" {
		line.ID = entry.InsertID
	}
	if !entry.Timestamp.IsZero() {
		line.FirstObserved = entry.Timestamp
	}
	if entry.Severity != logging.Default {
		line.Severity = entry.Severity.String()
	}
	if entry.LogName != "" {
		line.Source = entry.LogName
	}

	switch payload := entry.Payload.(type) {
	case string:
		line.Message = payload
	default:
		payloadMap, err := collections.ToJSONMap(payload)
		if err != nil {
			return nil
		}

		for k, v := range payloadMap {
			if err := logs.MapFieldToLogLine(k, v, line, mappingConfig); err != nil {
				continue
			}
		}
	}

	if entry.Resource != nil && entry.Resource.Labels != nil {
		for k, v := range entry.Resource.Labels {
			resourceKey := "resource." + k
			if err := logs.MapFieldToLogLine(resourceKey, v, line, mappingConfig); err != nil {
				continue
			}
		}
	}

	if entry.Labels != nil {
		for k, v := range entry.Labels {
			labelKey := "label." + k
			if err := logs.MapFieldToLogLine(labelKey, v, line, mappingConfig); err != nil {
				continue
			}
		}
	}

	line.SetHash()
	return line
}

// convertAPILogEntryToLogLine converts a loggingapi.LogEntry to a LogLine
func (gcp *cloudLogging) convertAPILogEntryToLogLine(entry *loggingapi.LogEntry, mappingConfig logs.FieldMappingConfig) *logs.LogLine {
	line := &logs.LogLine{
		Count: 1,
	}

	if entry.InsertId != "" {
		line.ID = entry.InsertId
	}
	if entry.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
			line.FirstObserved = ts
		}
	}
	if entry.Severity != "" {
		line.Severity = entry.Severity
	}
	if entry.LogName != "" {
		line.Source = entry.LogName
	}

	// Handle different payload types
	var payloadMap map[string]any
	if entry.TextPayload != "" {
		line.Message = entry.TextPayload
	} else if len(entry.JsonPayload) > 0 {
		payloadMap = make(map[string]any)
		if err := json.Unmarshal(entry.JsonPayload, &payloadMap); err == nil {
			if msg, ok := payloadMap["message"].(string); ok {
				line.Message = msg
			}
		}
	} else if len(entry.ProtoPayload) > 0 {
		payloadMap = make(map[string]any)
		json.Unmarshal(entry.ProtoPayload, &payloadMap)
	}

	// Map payload fields
	if payloadMap != nil {
		for k, v := range payloadMap {
			if err := logs.MapFieldToLogLine(k, v, line, mappingConfig); err != nil {
				continue
			}
		}
	}

	// Map resource labels
	if entry.Resource != nil && entry.Resource.Labels != nil {
		for k, v := range entry.Resource.Labels {
			resourceKey := "resource." + k
			if err := logs.MapFieldToLogLine(resourceKey, v, line, mappingConfig); err != nil {
				continue
			}
		}
	}

	// Map entry labels
	if entry.Labels != nil {
		for k, v := range entry.Labels {
			labelKey := "label." + k
			if err := logs.MapFieldToLogLine(labelKey, v, line, mappingConfig); err != nil {
				continue
			}
		}
	}

	line.SetHash()
	return line
}

// defaultFieldMappingConfig can vary widely depending on the source of the log.
//
//	Example: Audit logs have very different fields than a kubernetes pod log.
var defaultFieldMappingConfig = logs.FieldMappingConfig{
	Message:   []string{"message", "msg", "textPayload"},
	Timestamp: []string{"timestamp", "@timestamp", "lastTimestamp"},
	Severity:  []string{"severity", "level"},
	Source:    []string{"logName", "source"},
	Host:      []string{"resource.instance_id", "resource.pod_name", "resource.container_name"},
}
