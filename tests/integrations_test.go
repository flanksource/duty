package tests

import (
	"github.com/flanksource/duty/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration view", func() {
	It("should be able to call integrations view", func() {
		var integrations []models.Integration
		err := DefaultContext.DB().Find(&integrations).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(len(integrations)).To(BeNumerically(">", 0))
	})
})
