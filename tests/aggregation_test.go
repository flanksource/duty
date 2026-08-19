package tests

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

var _ = Describe("Aggregation", func() {
	It("rejects config cost aggregation and filtering", func() {
		_, err := query.Aggregate(DefaultContext, "configs", types.AggregatedResourceSelector{
			Aggregates: []types.AggregationField{{Function: "SUM", Field: "cost_total_30d", Alias: "total_cost"}},
		})
		Expect(err).To(MatchError(ContainSubstring("aggregation field 'cost_total_30d' is not allowed")))

		_, err = query.Aggregate(DefaultContext, "configs", types.AggregatedResourceSelector{
			ResourceSelector: types.ResourceSelector{Search: "cost_total_30d>0"},
			Aggregates:       []types.AggregationField{{Function: "COUNT", Field: "*", Alias: "count"}},
		})
		Expect(err).To(MatchError(ContainSubstring("query for column:cost_total_30d")))
	})

	type testCase struct {
		name           string
		selector       types.AggregatedResourceSelector
		expectedResult []types.AggregateRow
	}

	DescribeTable("should aggregate resources correctly",
		func(tc testCase) {
			results, err := query.Aggregate(DefaultContext, "configs", tc.selector)
			Expect(err).ToNot(HaveOccurred())

			Expect(results).To(Equal(tc.expectedResult))
		},
		Entry("count resources by type", testCase{
			name: "count resources by type",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types:  []string{"Kubernetes::Pod", "Kubernetes::Node", "Kubernetes::Deployment"},
					Search: "@order=type",
				},
				GroupBy: []string{"type"},
				Aggregates: []types.AggregationField{
					{Function: "COUNT", Field: "*", Alias: "count"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"type": "Kubernetes::Deployment", "count": int64(3)},
				{"type": "Kubernetes::Node", "count": int64(3)},
				{"type": "Kubernetes::Pod", "count": int64(3)},
			},
		}),
		Entry("group by cluster", testCase{
			name: "group by cluster",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types:  []string{"Kubernetes::Node"},
					Search: "@order=cluster",
				},
				GroupBy: []string{"tags.cluster"},
				Aggregates: []types.AggregationField{
					{Function: "COUNT", Field: "*", Alias: "count"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"cluster": "aws", "count": int64(2)},
				{"cluster": "demo", "count": int64(1)},
			},
		}),
		Entry("calculate MIN created_at by type", testCase{
			name: "calculate MIN created_at by type",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types:  []string{"Kubernetes::Pod", "Kubernetes::Node"},
					Search: "@order=type",
				},
				GroupBy: []string{"type"},
				Aggregates: []types.AggregationField{
					{Function: "MIN", Field: "created_at", Alias: "earliest"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"type": "Kubernetes::Node", "earliest": dummy.DummyCreatedAt.In(time.Local)},
				{"type": "Kubernetes::Pod", "earliest": dummy.DummyCreatedAt.In(time.Local)},
			},
		}),
		Entry("healthy deployments for piechart", testCase{
			name: "combine multiple aggregation functions",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types: []string{"Kubernetes::Deployment"},
				},
				GroupBy: []string{"health"},
				Aggregates: []types.AggregationField{
					{Function: "COUNT", Field: "*", Alias: "total"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"health": "healthy", "total": int64(3)},
			},
		}),
	)
})
