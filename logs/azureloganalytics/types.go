// ABOUTME: Defines request types for querying Azure Monitor Log Analytics workspaces.
// ABOUTME: Uses KQL (Kusto Query Language) queries against Log Analytics workspace IDs.
package azureloganalytics

import (
	"github.com/flanksource/duty/logs"
)

// Request represents parameters for Azure Log Analytics queries.
//
// +kubebuilder:object:generate=true
type Request struct {
	logs.LogsRequestBase `json:",inline" template:"true"`

	// WorkspaceID is the Azure Log Analytics workspace ID to query.
	WorkspaceID string `json:"workspaceID" template:"true"`

	// Query is the KQL (Kusto Query Language) query to execute.
	Query string `json:"query" template:"true"`
}
