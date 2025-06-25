package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
)

var _ = Describe("buildJSONFieldSelector", func() {
	DescribeTable("should generate correct SQL selector and alias for JSON field access",
		func(field, expectedSelector, expectedAlias string) {
			selector, alias := query.BuildJSONFieldSelector(field)
			Expect(selector).To(Equal(expectedSelector))
			Expect(alias).To(Equal(expectedAlias))
		},
		Entry("simple tags field", "tags.cluster", `tags->>'cluster'`, "cluster"),
		Entry("simple labels field", "labels.env", `labels->>'env'`, "env"),
		Entry("simple config field", "config.version", `config->>'version'`, "version"),
		Entry("nested config field", "config.author.name", `config->'author'->>'name'`, "author_name"),
		Entry("deeply nested config field", "config.metadata.labels.app", `config->'metadata'->'labels'->>'app'`, "metadata_labels_app"),
		Entry("properties field", "properties.cost", `jsonb_path_query_first(properties, '$.cost')`, "cost"),
		Entry("nested properties field", "properties.metrics.cpu", `jsonb_path_query_first(properties, '$.metrics.cpu')`, "metrics_cpu"),
		Entry("custom JSON column", "custom.field", `custom->>'field'`, "field"),
		Entry("nested custom JSON column", "custom.nested.value", `custom->'nested'->>'value'`, "nested_value"),
		Entry("non-JSON field", "name", "name", ""),
	)
})
