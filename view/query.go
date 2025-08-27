package view

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/dataquery"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type Query struct {
	dataquery.Query `json:",inline" yaml:",inline" template:"true"`

	// Configs queries config items
	Configs *types.ResourceSelector `json:"configs,omitempty" yaml:"configs,omitempty" template:"true"`

	// Changes queries config changes
	Changes *types.ResourceSelector `json:"changes,omitempty" yaml:"changes,omitempty" template:"true"`
}

func (v *Query) IsEmpty() bool {
	return v.Configs == nil && v.Changes == nil && v.Query.IsEmpty()
}

// ExecuteQuery executes a single query and returns results with query name
func ExecuteQuery(ctx context.Context, q Query) ([]dataquery.QueryResultRow, error) {
	var results []dataquery.QueryResultRow
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
	} else {
		return dataquery.ExecuteQuery(ctx, q.Query)
	}

	return results, nil
}
