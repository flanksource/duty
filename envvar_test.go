package duty

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test GetSecretFromCache using a fake kubernetes client

var _ = Describe("EnvVar", func() {
	It("should lookup kubernetes secrets", func() {
		val, err := GetConfigMapFromCache(testClient, "default", "test-cm", "foo")
		Expect(err).ToNot(HaveOccurred())
		Expect(val).To(Equal("bar"))
	})

	It("should lookup configmaps", func() {
		val, err := GetSecretFromCache(testClient, "default", "test-secret", "foo")
		Expect(err).ToNot(HaveOccurred())
		Expect(val).To(Equal("secret"))
	})

})
