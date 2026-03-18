package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/gomplate/v3"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

type catalogQueryCase struct {
	Name       string         `yaml:"name"`
	Expression string         `yaml:"expression"`
	Args       map[string]any `yaml:"args"`
	Assertions []string       `yaml:"assertions"`
}

type catalogQueryFixture struct {
	Cases []catalogQueryCase `yaml:"cases"`
}

var _ = ginkgo.Describe("catalog.query", ginkgo.Ordered, func() {
	ginkgo.BeforeAll(func() {
		err := query.SyncConfigCache(DefaultContext)
		Expect(err).ToNot(HaveOccurred())
	})

	fixtures, err := filepath.Glob("testdata/catalog_query/*.yaml")
	if err != nil {
		panic(err)
	}

	for _, fixturePath := range fixtures {
		data, err := os.ReadFile(fixturePath)
		if err != nil {
			panic(err)
		}

		var fixture catalogQueryFixture
		if err := yaml.Unmarshal(data, &fixture); err != nil {
			panic(err)
		}

		for _, tc := range fixture.Cases {
			tc := tc
			ginkgo.It(tc.Name, func() {
				for _, assertion := range tc.Assertions {
					// Inline the expression into the assertion so it runs as one CEL evaluation.
					combined := strings.ReplaceAll(assertion, "result", tc.Expression)
					ok, err := DefaultContext.RunTemplateBool(gomplate.Template{Expression: combined}, tc.Args)
					Expect(err).ToNot(HaveOccurred(), "CEL error in %q: %s", tc.Name, assertion)
					Expect(ok).To(BeTrue(), "assertion failed in %q: %s\ncombined: %s", tc.Name, assertion, fmt.Sprintf("=> %s", combined))
				}
			})
		}
	}
})
