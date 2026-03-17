// ABOUTME: Implements log searching against Azure Monitor Log Analytics workspaces.
// ABOUTME: Executes KQL queries and maps tabular results to canonical LogLine format.
package azureloganalytics

import (
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/flanksource/commons/utils"
	"github.com/samber/lo"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
)

// Searcher implements log searching using Azure Log Analytics.
type Searcher struct {
	conn          connection.AzureConnection
	mappingConfig *logs.FieldMappingConfig
}

// New creates a new Azure Log Analytics searcher.
func New(conn connection.AzureConnection, mappingConfig *logs.FieldMappingConfig) *Searcher {
	return &Searcher{
		conn:          conn,
		mappingConfig: mappingConfig,
	}
}

// Search executes a KQL query against an Azure Log Analytics workspace.
func (s *Searcher) Search(ctx context.Context, request Request) (*logs.LogResult, error) {
	if request.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if request.WorkspaceID == "" {
		return nil, fmt.Errorf("workspaceID is required")
	}

	if err := s.conn.HydrateConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to hydrate connection: %w", err)
	}

	credential, err := s.conn.TokenCredential()
	if err != nil {
		return nil, fmt.Errorf("failed to create token credential: %w", err)
	}

	client, err := azquery.NewLogsClient(credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs client: %w", err)
	}

	body := azquery.Body{
		Query: &request.Query,
	}

	if request.Start != "" || request.End != "" {
		timespan, err := buildTimespan(request)
		if err != nil {
			return nil, fmt.Errorf("failed to build timespan: %w", err)
		}
		body.Timespan = timespan
	}

	var maxLogLines int
	if request.Limit != "" {
		limit, err := strconv.ParseInt(request.Limit, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		maxLogLines = int(limit)
	}

	resp, err := client.QueryWorkspace(ctx, request.WorkspaceID, body, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("query returned error: %s", resp.Error.Error())
	}

	mappingConfig := DefaultFieldMappingConfig
	if s.mappingConfig != nil {
		mappingConfig = s.mappingConfig.WithDefaults(DefaultFieldMappingConfig)
	}

	result := &logs.LogResult{
		Logs:     make([]*logs.LogLine, 0),
		Metadata: make(map[string]any),
	}

	for _, table := range resp.Tables {
		if table == nil {
			continue
		}

		columnNames := make([]string, len(table.Columns))
		for i, col := range table.Columns {
			if col.Name != nil {
				columnNames[i] = *col.Name
			}
		}

		result.Metadata["totalRows"] = len(table.Rows)

		for _, row := range table.Rows {
			line := &logs.LogLine{
				Count: 1,
			}

			for i, value := range row {
				if i >= len(columnNames) || value == nil {
					continue
				}

				if err := logs.MapFieldToLogLine(columnNames[i], value, line, mappingConfig); err != nil {
					return nil, fmt.Errorf("failed to map field %s: %w", columnNames[i], err)
				}
			}

			if line.Message == "" {
				if m, err := utils.Stringify(row); err == nil {
					line.Message = m
				}
			}

			line.SetHash()
			result.Logs = append(result.Logs, line)

			if maxLogLines > 0 && len(result.Logs) >= maxLogLines {
				break
			}
		}

		if maxLogLines > 0 && len(result.Logs) >= maxLogLines {
			break
		}
	}

	logs.GroupLogs(result, mappingConfig)
	return result, nil
}

// buildTimespan constructs an ISO8601 time interval from the request's start/end fields.
func buildTimespan(request Request) (*azquery.TimeInterval, error) {
	start, err := request.GetStart()
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}

	end, err := request.GetEnd()
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}

	timespan := azquery.NewTimeInterval(start, end)
	return lo.ToPtr(timespan), nil
}

// DefaultFieldMappingConfig defines sensible defaults for common Azure Monitor log columns.
var DefaultFieldMappingConfig = logs.FieldMappingConfig{
	Timestamp: []string{"TimeGenerated", "Timestamp"},
	Message:   []string{"Message", "RenderedDescription", "ResultDescription"},
	Severity:  []string{"SeverityLevel", "Level", "EventLevelName"},
	Host:      []string{"Computer", "_ResourceId"},
	Source:    []string{"Source", "Category", "OperationName"},
}
