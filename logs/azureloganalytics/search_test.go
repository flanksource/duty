// ABOUTME: Manual integration test for Azure Log Analytics searcher.
package azureloganalytics

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

func TestSearch_Manual(t *testing.T) {
	t.Skip("Manual test - requires Azure credentials and workspace ID")

	g := NewWithT(t)
	ctx := context.New()

	conn := connection.AzureConnection{
		ClientID:     &types.EnvVar{ValueStatic: ""},
		ClientSecret: &types.EnvVar{ValueStatic: ""},
		TenantID:     "",
	}

	searcher := New(conn, nil)

	request := Request{
		WorkspaceID: "",
		Query:       "AzureActivity | top 5 by TimeGenerated",
	}

	result, err := searcher.Search(ctx, request)
	g.Expect(err).To(BeNil())
	g.Expect(result).ToNot(BeNil())
	g.Expect(len(result.Logs)).To(BeNumerically(">", 0))
}
