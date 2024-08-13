package openapi

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlaybookSpec", func() {
	var (
		data []byte
		err  error
	)

	DescribeTable("Validate",
		func(spec string, invalid bool) {
			data, err = os.ReadFile(filepath.Join("testdata", spec))
			Expect(err).ToNot(HaveOccurred())

			validationError, err := ValidatePlaybookSpec(data)
			Expect(err).ToNot(HaveOccurred())
			if invalid {
				Expect(validationError).To(HaveOccurred())
			} else {
				Expect(validationError).ToNot(HaveOccurred())
			}
		},
		Entry("invalid playbook", "playbook-invalid.json", true),
		Entry("valid playbook", "playbook-valid.json", false),
	)
})

func TestOpenAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenAPI Suite")
}
