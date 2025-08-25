package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

var _ = Describe("Aggregation", func() {
	type testCase struct {
		name           string
		selector       types.AggregatedResourceSelector
		expectedResult []types.AggregateRow
	}

	DescribeTable("should aggregate resources correctly",
		func(tc testCase) {
			results, err := query.Aggregate(DefaultContext, "config_items", tc.selector)
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
					{Function: "MIN", Field: "cost_total_30d", Alias: "cheapest"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"type": "Kubernetes::Node", "cheapest": fmt.Sprintf("%.4f", dummy.KubernetesNodeA.CostTotal30d)},
				{"type": "Kubernetes::Pod", "cheapest": fmt.Sprintf("%.4f", dummy.LogisticsAPIPodConfig.CostTotal30d)},
			},
		}),
		Entry("calculate MAX created_at by type", testCase{
			name: "calculate MAX created_at by type",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types:  []string{"Kubernetes::Pod", "Kubernetes::Node"},
					Search: "@order=type",
				},
				GroupBy: []string{"type"},
				Aggregates: []types.AggregationField{
					{Function: "MAX", Field: "cost_total_30d", Alias: "most_expensive"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"type": "Kubernetes::Node", "most_expensive": fmt.Sprintf("%.4f", dummy.KubernetesNodeAKSPool1.CostTotal30d)},
				{"type": "Kubernetes::Pod", "most_expensive": fmt.Sprintf("%.4f", dummy.LogisticsAPIPodConfig.CostTotal30d)},
			},
		}),
		Entry("calculate SUM created_at by type", testCase{
			name: "calculate SUM created_at by type",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types:  []string{"Kubernetes::Pod", "Kubernetes::Node"},
					Search: "@order=type",
				},
				GroupBy: []string{"type"},
				Aggregates: []types.AggregationField{
					{Function: "SUM", Field: "cost_total_30d", Alias: "total_cost"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"type": "Kubernetes::Node", "total_cost": fmt.Sprintf("%.4f", dummy.KubernetesNodeAKSPool1.CostTotal30d+dummy.KubernetesNodeB.CostTotal30d+dummy.KubernetesNodeA.CostTotal30d)},
				{"type": "Kubernetes::Pod", "total_cost": fmt.Sprintf("%.4f", dummy.LogisticsAPIPodConfig.CostTotal30d)},
			},
		}),
		Entry("combine multiple aggregation functions", testCase{
			name: "combine multiple aggregation functions",
			selector: types.AggregatedResourceSelector{
				ResourceSelector: types.ResourceSelector{
					Types:  []string{"Kubernetes::Node"},
					Search: "@order=most_expensive",
				},
				GroupBy: []string{"tags.cluster"},
				Aggregates: []types.AggregationField{
					{Function: "COUNT", Field: "*", Alias: "total_count"},
					{Function: "MIN", Field: "cost_total_30d", Alias: "cheapest"},
					{Function: "MAX", Field: "cost_total_30d", Alias: "most_expensive"},
				},
			},
			expectedResult: []types.AggregateRow{
				{"cluster": "aws", "total_count": int64(2), "cheapest": fmt.Sprintf("%.4f", dummy.KubernetesNodeA.CostTotal30d), "most_expensive": fmt.Sprintf("%.4f", dummy.KubernetesNodeB.CostTotal30d)},
				{"cluster": "demo", "total_count": int64(1), "cheapest": fmt.Sprintf("%.4f", dummy.KubernetesNodeAKSPool1.CostTotal30d), "most_expensive": fmt.Sprintf("%.4f", dummy.KubernetesNodeAKSPool1.CostTotal30d)},
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
