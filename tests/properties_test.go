package tests

import (
	"time"

	"github.com/flanksource/duty/context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Properties", Ordered, func() {
	BeforeAll(func() {
		Expect(DefaultContext.DB().Exec("TRUNCATE properties").Error).ToNot(HaveOccurred())
	})
	It("Should save properties to db", func() {
		err := context.UpdateProperties(DefaultContext, map[string]string{
			"john":  "doe",
			"hello": "world",
		})
		Expect(err).ToNot(HaveOccurred())

		props := DefaultContext.Properties()
		Expect(props.String("john", "")).To(Equal("doe"))
		Expect(props.String("hello", "")).To(Equal("world"))
		Expect(props.String("hello1", "")).To(BeEmpty())
	})

	It("Should default int values", func() {
		Expect(DefaultContext.Properties().Int("property1", 10)).To(Equal(10))
		Expect(context.UpdateProperty(DefaultContext, "property1", "20")).Error().ToNot(HaveOccurred())
		Expect(DefaultContext.Properties().Int("property1", 10)).To(Equal(20))
	})

	It("Should default duration values", func() {
		Expect(DefaultContext.Properties().Duration("duration1", 1*time.Minute)).To(Equal(1 * time.Minute))
		Expect(context.UpdateProperty(DefaultContext, "duration1", "5m")).Error().ToNot(HaveOccurred())
		Expect(DefaultContext.Properties().Duration("duration1", 1*time.Minute)).To(Equal(5 * time.Minute))
	})
})
