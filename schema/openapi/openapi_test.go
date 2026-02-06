package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/types"
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

var _ = Describe("GenerateSchema", func() {
	It("adds Go comments to schema descriptions", func() {
		schema, err := GenerateSchema(&types.ResourceSelector{})
		Expect(err).ToNot(HaveOccurred())

		var schemaJSON map[string]any
		Expect(json.Unmarshal(schema, &schemaJSON)).To(Succeed())

		defs, ok := schemaJSON["$defs"].(map[string]any)
		Expect(ok).To(BeTrue())

		resourceSelectorDef, ok := defs["ResourceSelector"].(map[string]any)
		Expect(ok).To(BeTrue())

		properties, ok := resourceSelectorDef["properties"].(map[string]any)
		Expect(ok).To(BeTrue())

		agentField, ok := properties["agent"].(map[string]any)
		Expect(ok).To(BeTrue())

		agentDescription, ok := agentField["description"].(string)
		Expect(ok).To(BeTrue())
		Expect(agentDescription).To(ContainSubstring("Agent can be the agent id or the name of the agent"))
	})
})

func TestOpenAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenAPI Suite")
}
