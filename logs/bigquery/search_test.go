package bigquery

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

// Simple integration test that can be run manually for debugging
func TestSearch_Manual(t *testing.T) {
	t.Skip("Manual test")

	g := NewWithT(t)
	ctx := context.New().WithNamespace("mc")

	conn := connection.GCPConnection{
		Project: "workload-prod-eu-02",
		Credentials: &types.EnvVar{
			ValueFrom: &types.EnvVarSource{
				SecretKeyRef: &types.SecretKeySelector{
					Key: "credentials",
					LocalObjectReference: types.LocalObjectReference{
						Name: "gcloud-flanksource",
					},
				},
			},
		},
	}

	searcher := New(conn, nil)
	defer searcher.Close()

	request := Request{
		Query: "SELECT name, number, gender FROM `bigquery-public-data.usa_names.usa_1910_2013` " +
			"WHERE state = \"TX\" " +
			"LIMIT 5",
	}

	result, err := searcher.Search(ctx, request)
	g.Expect(err).To(BeNil())
	g.Expect(len(result.Logs)).To(Equal(5))
	for _, log := range result.Logs {
		g.Expect(log.Labels["name"]).To(Not(BeEmpty()))
		g.Expect(log.Labels["number"]).To(Not(BeEmpty()))
		g.Expect(log.Labels["gender"]).To(Not(BeEmpty()))
	}
}
