package tests

import (
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = ginkgo.Describe("Config Location Functions", ginkgo.Ordered, func() {
	ginkgo.Context("get_children_by_location function", func() {
		ginkgo.It("should find children based on external_id without prefix filter", func() {
			results, err := query.FindConfigChildrenByLocation(DefaultContext, dummy.EKSCluster.ID, "", false)
			Expect(err).To(BeNil())

			resultIDs := lo.Map(results, func(r query.ConfigChildrenByLocation, _ int) uuid.UUID { return r.ID })
			Expect(resultIDs).To(ContainElements([]uuid.UUID{dummy.EKSCluster.ID, dummy.KubernetesNodeA.ID}))
		})
	})
})
