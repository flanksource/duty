package gcpcloudlogging

import (
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/logging/logadmin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
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

type cloudLogging struct {
	client        *logadmin.Client
	mappingConfig *logs.FieldMappingConfig
}

func (gcp *cloudLogging) Close() error {
	return gcp.client.Close()
}

func New(ctx context.Context, conn connection.GCPConnection, mappingConfig *logs.FieldMappingConfig) (*cloudLogging, error) {
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

	return &cloudLogging{
		client:        adminClient,
		mappingConfig: mappingConfig,
	}, nil
}

func (gcp *cloudLogging) Search(ctx context.Context, request Request) (*logs.LogResult, error) {
	var maxLogLines = 1000
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
