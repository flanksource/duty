package duty

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/models"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// For debugging
// nolint
func prettytree(mytree []*models.Component) {
	for _, c := range mytree {
		fmt.Printf("- %s {analysis: %v}\n\n", c.Name, c.Summary)
		for _, cc := range c.Components {
			fmt.Printf("  |- %s {analysis: %v}\n\n", cc.Name, cc.Summary)
			for _, ccc := range cc.Components {
				fmt.Printf("    |- %s {analysis: %v}\n\n", ccc.Name, ccc.Summary)
				for _, cccc := range ccc.Components {
					fmt.Printf("      |- %s {analysis: %v}\n\n", cccc.Name, cccc.Summary)
				}
			}
		}
	}
}

func testTopologyJSON(opts TopologyOptions, path string) {
	tree, err := QueryTopology(testDBPGPool, opts)
	Expect(err).ToNot(HaveOccurred())

	treeJSON, err := json.Marshal(tree)
	Expect(err).ToNot(HaveOccurred())

	expected := readTestFile(path)
	jqExpr := `del(.. | .created_at?, .updated_at?)`
	matchJSON([]byte(expected), treeJSON, &jqExpr)
}

var _ = ginkgo.Describe("Topology behavior", ginkgo.Ordered, func() {

	ginkgo.It("Should create root tree", func() {
		testTopologyJSON(TopologyOptions{}, "fixtures/expectations/topology_root_tree.json")
	})

	ginkgo.It("Should create child tree", func() {
		testTopologyJSON(TopologyOptions{ID: dummy.NodeA.ID.String()}, "fixtures/expectations/topology_child_tree.json")
	})

	ginkgo.It("Should test depth 1 root tree", func() {
		testTopologyJSON(TopologyOptions{Depth: 1}, "fixtures/expectations/topology_depth_1_root_tree.json")
	})

	ginkgo.It("Should test depth 2 root tree", func() {
		testTopologyJSON(TopologyOptions{Depth: 2}, "fixtures/expectations/topology_depth_2_root_tree.json")
	})

	ginkgo.It("Should test depth 1 tree child tree", func() {
		testTopologyJSON(TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Depth: 1}, "fixtures/expectations/topology_depth_1_child_tree.json")
	})

	ginkgo.It("Should test depth 2 tree child tree", func() {
		// TODO: Current query with a component_id defined does not return the children if
		// they are linked via parent_id of that component
		ginkgo.Skip("SQL Query needs to be fixed for this to work")

		testTopologyJSON(TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Depth: 1}, "fixtures/expectations/topology_depth_2_child_tree.json")
	})

	ginkgo.It("Should test tree with labels", func() {
		testTopologyJSON(TopologyOptions{Labels: map[string]string{"telemetry": "enabled"}}, "fixtures/expectations/topology_tree_with_label_filter.json")
	})

	ginkgo.It("Should test tree with owner", func() {
		testTopologyJSON(TopologyOptions{Owner: "logistics-team"}, "fixtures/expectations/topology_tree_with_owner_filter.json")
	})

	ginkgo.It("Should test tree with type filter", func() {
		testTopologyJSON(TopologyOptions{Types: []string{"Entity"}}, "fixtures/expectations/topology_tree_with_type_filter.json")
	})

	ginkgo.It("Should test tree with negative type filter", func() {
		// TODO: Change implementation of matchItems to fix this
		ginkgo.Skip("Current implementation does not filter negative types correctly")

		testTopologyJSON(TopologyOptions{Types: []string{"!KubernetesCluster"}}, "fixtures/expectations/topology_tree_with_negative_type_filter.json")
	})

	ginkgo.It("Should test tree with status filter", func() {
		testTopologyJSON(TopologyOptions{Status: []string{string(models.ComponentStatusWarning)}}, "fixtures/expectations/topology_tree_with_status_filter.json")
	})

	ginkgo.It("Should test tree with ID and status filter", func() {
		testTopologyJSON(TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Status: []string{string(models.ComponentStatusHealthy)}}, "fixtures/expectations/topology_tree_with_id_and_status_filter.json")
	})

})

func Test_populateTopologyResult(t *testing.T) {
	type args struct {
		components models.Components
		res        TopologyResponse
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "first",
			args: args{
				res: TopologyResponse{
					Types:          []string{"type-1", "type-1-1", "duplicate-type", "type-2", "type-2-1", "type-2-1-1", "type-2-1-2", "type-2-1-3", "type-2-2"},
					HealthStatuses: []string{"OK", "UNHEALTHY"},
					Tags: []Tag{
						{Key: "tag", Val: "tag-1"},
						{Key: "tag", Val: "tag-1-1"},
						{Key: "tag", Val: "duplicate"},
						{Key: "tag", Val: "tag-2"},
						{Key: "tag", Val: "tag-2-1"},
						{Key: "tag", Val: "tag-2-1-1"},
						{Key: "tag", Val: "tag-2-1-2"},
						{Key: "tag", Val: "tag-2-1-3"},
						{Key: "tag", Val: "tag-2-2"},
					},
				},
				components: models.Components{
					{
						Name:   "first",
						Status: "OK",
						Type:   "type-1",
						Labels: map[string]string{
							"tag": "tag-1",
						},
						Components: models.Components{
							{
								Name:   "first-first",
								Status: "OK",
								Type:   "type-1-1",
								Labels: map[string]string{
									"tag": "tag-1-1",
								},
							},
							{
								Name:   "first-second",
								Status: "OK",
								Type:   "duplicate-type",
								Labels: map[string]string{
									"tag": "duplicate",
								},
							},
							{
								Name:   "first-third",
								Status: "OK",
								Type:   "duplicate-type",
								Labels: map[string]string{
									"tag": "duplicate",
								},
							},
						},
					},
					{
						Name:   "second",
						Status: "OK",
						Type:   "type-2",
						Labels: map[string]string{
							"tag": "tag-2",
						},
						Components: models.Components{
							{
								Name:   "second-first",
								Status: "OK",
								Type:   "type-2-1",
								Labels: map[string]string{
									"tag": "tag-2-1",
								},
								Components: models.Components{
									{
										Name:   "second-first-first",
										Status: "OK",
										Type:   "type-2-1-1",
										Labels: map[string]string{
											"tag": "tag-2-1-1",
										},
									},
									{
										Name:   "second-first-second",
										Status: "OK",
										Type:   "type-2-1-2",
										Labels: map[string]string{
											"tag": "tag-2-1-2",
										},
									},
									{
										Name:   "second-first-third",
										Status: "OK",
										Type:   "type-2-1-3",
										Labels: map[string]string{
											"tag": "tag-2-1-3",
										},
									},
								},
							},
							{
								Name:   "second-second",
								Status: "UNHEALTHY",
								Type:   "type-2-2",
								Labels: map[string]string{
									"tag": "tag-2-2",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res TopologyResponse
			addMetadata(tt.args.components, &res)
			if diff := cmp.Diff(tt.args.res, res, cmpopts.IgnoreUnexported(TopologyResponse{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
