package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = ginkgo.Describe("Config Location Functions", func() {
	ginkgo.It("should find children by location using find_children_by_location function", func() {
		results, err := query.FindConfigChildrenByLocation(DefaultContext, dummy.LogisticsAPIDeployment.ID, false)
		Expect(err).To(BeNil())
		Expect(results).To(HaveLen(2))

		expected := []query.ConfigChildrenByLocation{
			{ID: dummy.LogisticsAPIReplicaSet.ID, Type: *dummy.LogisticsAPIReplicaSet.Type, Name: *dummy.LogisticsAPIReplicaSet.Name},
			{ID: dummy.LogisticsAPIPodConfig.ID, Type: *dummy.LogisticsAPIPodConfig.Type, Name: *dummy.LogisticsAPIPodConfig.Name},
		}
		Expect(results).To(ConsistOf(expected))
	})
})
