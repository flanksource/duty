package duty

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/testutils"
	"github.com/flanksource/duty/types"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
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
	tree, err := QueryTopology(context.Background(), testutils.TestDBPGPool, opts)
	Expect(err).ToNot(HaveOccurred())

	treeJSON, err := json.Marshal(tree)
	Expect(err).ToNot(HaveOccurred())

	expected := readTestFile(path)

	jqExpr := `del(.. | .created_at?, .updated_at?)`
	matchJSON([]byte(expected), treeJSON, &jqExpr)
}

var _ = ginkgo.Describe("Topology behavior", func() {
	format.MaxLength = 0 // So the diff is not truncated.

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
		// FIXME:
		ginkgo.Skip("type filter is applied on Go side. The team list is already populated by the SQL query and later the component might be removed by the type filter.")

		testTopologyJSON(TopologyOptions{Types: []string{"Entity"}}, "fixtures/expectations/topology_tree_with_type_filter.json")
	})

	ginkgo.It("Should test tree with negative type filter", func() {
		// TODO: Change implementation of matchItems to fix this
		ginkgo.Skip("Current implementation does not filter negative types correctly")

		testTopologyJSON(TopologyOptions{Types: []string{"!KubernetesCluster"}}, "fixtures/expectations/topology_tree_with_negative_type_filter.json")
	})

	ginkgo.It("Should test tree with status filter", func() {
		// FIXME:
		ginkgo.Skip("status filter is applied on Go side. The team list is already populated by the SQL query and later the component might be removed by the status filter.")

		testTopologyJSON(TopologyOptions{Status: []string{string(types.ComponentStatusWarning)}}, "fixtures/expectations/topology_tree_with_status_filter.json")
	})

	ginkgo.It("Should test tree with ID and status filter", func() {
		testTopologyJSON(TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Status: []string{string(types.ComponentStatusHealthy)}}, "fixtures/expectations/topology_tree_with_id_and_status_filter.json")
	})

	ginkgo.It("Should test tree with agent ID filter", func() {
		testTopologyJSON(TopologyOptions{AgentID: dummy.GCPAgent.ID.String()}, "fixtures/expectations/topology_tree_with_agent_id.json")
	})

	ginkgo.It("Should test tree with team filter", func() {
		testTopologyJSON(TopologyOptions{Team: dummy.PaymentTeam.Name}, "fixtures/expectations/topology_tree_with_team_filter.json")
	})
})
