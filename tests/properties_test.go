package tests

import (
	"github.com/flanksource/duty/context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Properties", func() {
	It("Should save properties to db", func() {
		err := context.UpdateProperties(DefaultContext, map[string]string{
			"john":  "doe",
			"hello": "world",
		})
		Expect(err).ToNot(HaveOccurred())

		retrieved := DefaultContext.Properties()
		Expect(retrieved).To(HaveKeyWithValue("john", "doe"))
		Expect(retrieved).To(HaveKeyWithValue("hello", "world"))
		Expect(retrieved).ToNot(HaveKey("hello1"))
	})
})
