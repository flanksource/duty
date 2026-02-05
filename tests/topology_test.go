package tests

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/tests/matcher"
	"github.com/flanksource/duty/types"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/samber/lo"
)

//lint:ignore U1000 For debugging
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

//lint:ignore U1000 For debugging
func writeJSONToFile(filepath string, data any) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath, b, 0644)
	if err != nil {
		return err
	}
	return nil
}

func testTopologyJSON(opts query.TopologyOptions, path string) {
	tree, err := query.Topology(DefaultContext, opts)
	Expect(err).ToNot(HaveOccurred())

	matcher.MatchFixture(path, tree, `del(.. | .created_at?, .updated_at?)`)
}

var _ = ginkgo.Describe("Topology", ginkgo.Pending, func() {
	format.MaxLength = 0 // Do not truncate diffs

	ginkgo.It("Should create root tree", func() {
		testTopologyJSON(query.TopologyOptions{}, "fixtures/expectations/topology_root_tree.json")
	})

	ginkgo.It("Should fetch minimal details of other children", func() {
		testTopologyJSON(query.TopologyOptions{ID: dummy.ClusterComponent.ID.String(), Depth: 5}, "fixtures/expectations/topology_cluster_component_tree.json")
	})

	ginkgo.It("Should create child tree", func() {
		testTopologyJSON(query.TopologyOptions{ID: dummy.NodeA.ID.String()}, "fixtures/expectations/topology_child_tree.json")
	})

	ginkgo.It("Should test depth 1 root tree", func() {
		testTopologyJSON(query.TopologyOptions{Depth: 1}, "fixtures/expectations/topology_depth_1_root_tree.json")
	})

	ginkgo.It("Should test depth 2 root tree", func() {
		testTopologyJSON(query.TopologyOptions{Depth: 2}, "fixtures/expectations/topology_depth_2_root_tree.json")
	})

	ginkgo.It("Should test depth 1 tree child tree", func() {
		testTopologyJSON(query.TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Depth: 1}, "fixtures/expectations/topology_depth_1_child_tree.json")
	})

	ginkgo.It("Should test depth 2 tree child tree", func() {
		// TODO: Current query with a component_id defined does not return the children if
		// they are linked via parent_id of that component
		ginkgo.Skip("SQL Query needs to be fixed for this to work")

		testTopologyJSON(query.TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Depth: 1}, "fixtures/expectations/topology_depth_2_child_tree.json")
	})

	ginkgo.It("Should test tree with labels", func() {
		testTopologyJSON(query.TopologyOptions{Labels: map[string]string{"telemetry": "enabled"}}, "fixtures/expectations/topology_tree_with_label_filter.json")
	})

	ginkgo.It("Should test tree with owner", func() {
		testTopologyJSON(query.TopologyOptions{Owner: "logistics-team"}, "fixtures/expectations/topology_tree_with_owner_filter.json")
	})

	ginkgo.It("Should test tree with type filter", func() {
		testTopologyJSON(query.TopologyOptions{Types: []string{"Entity"}}, "fixtures/expectations/topology_tree_with_type_filter.json")
	})

	ginkgo.It("Should test tree with negative type filter", func() {
		// TODO: Change implementation of matchItems to fix this
		ginkgo.Skip("Current implementation does not filter negative types correctly")

		testTopologyJSON(query.TopologyOptions{Types: []string{"!KubernetesCluster"}}, "fixtures/expectations/topology_tree_with_negative_type_filter.json")
	})

	ginkgo.It("Should test tree with status filter", func() {
		testTopologyJSON(query.TopologyOptions{Status: []string{string(types.ComponentStatusWarning)}}, "fixtures/expectations/topology_tree_with_status_filter.json")
	})

	ginkgo.It("Should test tree with ID and status filter", func() {
		testTopologyJSON(query.TopologyOptions{ID: dummy.LogisticsAPI.ID.String(), Status: []string{string(types.ComponentStatusHealthy)}}, "fixtures/expectations/topology_tree_with_id_and_status_filter.json")
	})

	ginkgo.It("Should test tree with agent ID filter", func() {
		testTopologyJSON(query.TopologyOptions{AgentID: dummy.GCPAgent.ID.String()}, "fixtures/expectations/topology_tree_with_agent_id.json")
	})

	ginkgo.It("Should test tree with sort options", func() {
		testTopologyJSON(query.TopologyOptions{ID: dummy.PodsComponent.ID.String(), SortBy: "field:memory"}, "fixtures/expectations/topology_tree_with_sort.json")

		testTopologyJSON(query.TopologyOptions{ID: dummy.PodsComponent.ID.String(), SortBy: "field:memory", SortOrder: "desc"}, "fixtures/expectations/topology_tree_with_desc_sort.json")
	})
})

var _ = ginkgo.Describe("Check topology sort", func() {
	components := models.Components{
		&models.Component{Name: "Zero", Properties: models.Properties{&models.Property{Name: "size", Value: nil}}},
		&models.Component{Name: "Highest", Properties: models.Properties{&models.Property{Name: "size", Value: lo.ToPtr(int64(50))}}},
		&models.Component{Name: "Lowest", Properties: models.Properties{&models.Property{Name: "size", Value: lo.ToPtr(int64(-5))}}},
		&models.Component{Name: "Medium", Properties: models.Properties{&models.Property{Name: "size", Value: lo.ToPtr(int64(0))}}},
		&models.Component{Name: "Zero", Properties: models.Properties{&models.Property{Name: "size", Value: nil}}},
	}

	ginkgo.It("Should sort components in ascending order", func() {
		query.SortComponentsByField(components, query.TopologyQuerySortBy("field:size"), true)
		Expect(components[0].Name).To(Equal("Lowest"))
		Expect(components[1].Name).To(Equal("Medium"))
		Expect(components[2].Name).To(Equal("Highest"))
		Expect(components[3].Name).To(Equal("Zero"))
		Expect(components[4].Name).To(Equal("Zero"))
	})

	ginkgo.It("Should sort components in descending order", func() {
		query.SortComponentsByField(components, query.TopologyQuerySortBy("field:size"), false)
		Expect(components[0].Name).To(Equal("Highest"))
		Expect(components[1].Name).To(Equal("Medium"))
		Expect(components[2].Name).To(Equal("Lowest"))
		Expect(components[3].Name).To(Equal("Zero"))
		Expect(components[4].Name).To(Equal("Zero"))
	})
})
