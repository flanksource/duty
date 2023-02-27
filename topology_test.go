package duty

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/models"
	_ "github.com/flanksource/duty/types"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTopology(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
}

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

var _ = ginkgo.Describe("Topology behavior", ginkgo.Ordered, func() {

	ginkgo.It("Should create root tree", func() {
		tree, err := QueryTopology(TopologyOptions{})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_root_tree.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should create child tree", func() {
		tree, err := QueryTopology(TopologyOptions{ID: dummy.NodeA.ID.String()})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_child_tree.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test depth 1 root tree", func() {
		tree, err := QueryTopology(TopologyOptions{Depth: 1})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_depth_1_root_tree.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test depth 2 root tree", func() {
		tree, err := QueryTopology(TopologyOptions{Depth: 2})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_depth_2_root_tree.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test depth 1 tree child tree", func() {
		tree, err := QueryTopology(TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Depth: 1})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_depth_1_child_tree.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test depth 2 tree child tree", func() {
		// TODO: Current query with a component_id defined does not return the children if
		// they are linked via parent_id of that component
		ginkgo.Skip("SQL Query needs to be fixed for this to work")
		tree, err := QueryTopology(TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Depth: 2})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_depth_2_child_tree.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test tree with labels", func() {
		tree, err := QueryTopology(TopologyOptions{Labels: map[string]string{"telemetry": "enabled"}})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_tree_with_label_filter.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test tree with owner", func() {
		tree, err := QueryTopology(TopologyOptions{Owner: "logistics-team"})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_tree_with_owner_filter.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test tree with type filter", func() {
		tree, err := QueryTopology(TopologyOptions{Types: []string{"Entity"}})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_tree_with_type_filter.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test tree with negative type filter", func() {
		// TODO: Change implementation of matchItems to fix this
		ginkgo.Skip("Current implementation does not filter negative types correctly")
		tree, err := QueryTopology(TopologyOptions{Types: []string{"!KubernetesCluster"}})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_tree_with_negative_type_filter.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})

	ginkgo.It("Should test tree with status filter", func() {
		tree, err := QueryTopology(TopologyOptions{Status: []string{string(models.ComponentStatusWarning)}})
		Expect(err).ToNot(HaveOccurred())

		treeJSON, err := json.Marshal(tree)
		Expect(err).ToNot(HaveOccurred())

		expected := readTestFile("fixtures/expectations/topology_tree_with_status_filter.json")
		Expect(expected).Should(MatchJSON(string(treeJSON)))
	})
})
