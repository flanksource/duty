package tests

import (
	"github.com/flanksource/duty/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubernetes", func() {
	It("should query resources", func() {
		Skip("Kubernetes cluster isn't available on CI")

		c, err := DefaultContext.Kubernetes()
		Expect(err).ToNot(HaveOccurred())

		objs, err := c.QueryResources(DefaultContext, types.ResourceSelector{Name: "default", Namespace: "", Types: []string{"Namespace"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(objs)).To(Equal(1))
	})
})
