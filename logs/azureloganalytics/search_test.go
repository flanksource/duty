// ABOUTME: Manual integration test for Azure Log Analytics searcher.
// ABOUTME: Requires valid Azure credentials and workspace ID to run.
package azureloganalytics

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/types"
)

func TestSearch_Manual(t *testing.T) {
	t.Skip("Manual test - requires Azure credentials and workspace ID")

	g := NewWithT(t)
	ctx := context.Background()

	conn := connection.AzureConnection{
		ClientID:     &types.EnvVar{ValueStatic: "your-client-id"},
		ClientSecret: &types.EnvVar{ValueStatic: "your-client-secret"},
		TenantID:     "your-tenant-id",
	}

	searcher := New(conn, nil)

	request := Request{
		WorkspaceID: "your-workspace-id",
		Query:       "AzureActivity | top 5 by TimeGenerated",
	}

	result, err := searcher.Search(ctx, request)
	g.Expect(err).To(BeNil())
	g.Expect(result).ToNot(BeNil())
	g.Expect(len(result.Logs)).To(BeNumerically(">", 0))
}
