package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/flanksource/duty/types"
	"github.com/onsi/gomega"
)

func TestGenerateChangeTypesSchema(t *testing.T) {
	g := gomega.NewWithT(t)

	outPath := filepath.Join(t.TempDir(), "change-types.schema.json")
	err := generateChangeTypesSchema(outPath)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	raw, err := os.ReadFile(outPath)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	var root struct {
		Schema string                     `json:"$schema"`
		ID     string                     `json:"$id"`
		Ref    string                     `json:"$ref"`
		Defs   map[string]json.RawMessage `json:"$defs"`
	}
	g.Expect(json.Unmarshal(raw, &root)).To(gomega.Succeed())

	g.Expect(root.Schema).To(gomega.Equal("https://json-schema.org/draft/2020-12/schema"))
	g.Expect(root.ID).To(gomega.Equal("https://github.com/flanksource/duty/types/config-change-details-schema"))
	g.Expect(root.Ref).To(gomega.Equal("#/$defs/ConfigChangeDetailsSchema"))

	g.Expect(root.Defs).To(gomega.HaveKey("ConfigChangeDetailsSchema"))
	g.Expect(root.Defs).ToNot(gomega.HaveKey("changeDetailsEnvelope"))
	g.Expect(root.Defs).ToNot(gomega.HaveKey("ChangeDetailsEnvelope"))

	expectedTypeNames := map[string]string{}
	for _, d := range types.ConfigChangeDetailTypes {
		name := reflect.TypeOf(d).Name()
		expectedTypeNames[name] = d.Kind()
	}
	for name := range expectedTypeNames {
		g.Expect(root.Defs).To(gomega.HaveKey(name), "missing $def for %s", name)
	}

	var rootDef struct {
		Description string            `json:"description"`
		OneOf       []map[string]any  `json:"oneOf"`
		Examples    []json.RawMessage `json:"examples"`
	}
	g.Expect(json.Unmarshal(root.Defs["ConfigChangeDetailsSchema"], &rootDef)).To(gomega.Succeed())
	g.Expect(rootDef.OneOf).To(gomega.HaveLen(len(types.ConfigChangeDetailTypes)))
	for _, entry := range rootDef.OneOf {
		g.Expect(entry).To(gomega.HaveKey("$ref"))
	}

	g.Expect(rootDef.Examples).To(gomega.HaveLen(len(types.ConfigChangeExamples)))
	for i, exampleRaw := range rootDef.Examples {
		var example map[string]any
		g.Expect(json.Unmarshal(exampleRaw, &example)).To(gomega.Succeed())
		kind, ok := example["kind"].(string)
		g.Expect(ok).To(gomega.BeTrue(), "example %d missing kind", i)
		g.Expect(kindSet(expectedTypeNames)).To(gomega.ContainElement(kind))
	}

	for name, expectedKind := range expectedTypeNames {
		var def struct {
			Required   []string                   `json:"required"`
			Properties map[string]json.RawMessage `json:"properties"`
		}
		g.Expect(json.Unmarshal(root.Defs[name], &def)).To(gomega.Succeed(), "unmarshal %s", name)
		g.Expect(def.Required).To(gomega.ContainElement("kind"), "%s missing kind in required", name)

		kindProp, ok := def.Properties["kind"]
		g.Expect(ok).To(gomega.BeTrue(), "%s missing kind property", name)
		var kindSchema struct {
			Type  string `json:"type"`
			Const string `json:"const"`
		}
		g.Expect(json.Unmarshal(kindProp, &kindSchema)).To(gomega.Succeed())
		g.Expect(kindSchema.Type).To(gomega.Equal("string"))
		g.Expect(kindSchema.Const).To(gomega.Equal(expectedKind), "%s kind const mismatch", name)
	}

	g.Expect(root.Defs).To(gomega.HaveKey("Event"))
}

func kindSet(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
