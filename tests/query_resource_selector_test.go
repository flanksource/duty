package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("SearchResourceSelectors", ginkgo.Ordered, func() {
	ginkgo.It("should find all 3 resources", func() {
		items, err := query.SearchResources(DefaultContext, query.SearchResourcesRequest{
			Configs:    []types.ResourceSelector{{ID: dummy.EKSCluster.ID.String()}},
			Components: []types.ResourceSelector{{ID: dummy.Logistics.ID.String()}},
			Checks:     []types.ResourceSelector{{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}},
		})
		Expect(err).To(BeNil())

		expectation := []query.SelectedResources{
			{ID: dummy.EKSCluster.ID.String(), Name: lo.FromPtr(dummy.EKSCluster.Name), Type: query.SelectedResourceTypeConfig},
			{
				ID:   dummy.LogisticsAPIHealthHTTPCheck.ID.String(),
				Icon: dummy.LogisticsAPIHealthHTTPCheck.Icon,
				Name: dummy.LogisticsAPIHealthHTTPCheck.Name,
				Type: query.SelectedResourceTypeCheck,
			},
			{ID: dummy.Logistics.ID.String(), Icon: dummy.Logistics.Icon, Name: dummy.Logistics.Name, Type: query.SelectedResourceTypeComponent},
		}
		Expect(items).To(ConsistOf(expectation))
	})
})
