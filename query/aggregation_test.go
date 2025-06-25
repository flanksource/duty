package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
)

var _ = Describe("buildJSONFieldSelector", func() {
	DescribeTable("should generate correct SQL for JSON field access",
		func(field, expected string) {
			result := query.BuildJSONFieldSelector(field)
			Expect(result).To(Equal(expected))
		},
		Entry("simple tags field", "tags.cluster", `tags->>'cluster' as "cluster"`),
		Entry("simple labels field", "labels.env", `labels->>'env' as "env"`),
		Entry("simple config field", "config.version", `config->>'version' as "version"`),
		Entry("nested config field", "config.author.name", `config->'author'->>'name' as "author_name"`),
		Entry("deeply nested config field", "config.metadata.labels.app", `config->'metadata'->'labels'->>'app' as "metadata_labels_app"`),
		Entry("properties field", "properties.cost", `jsonb_path_query_first(properties, '$.cost') as "cost"`),
		Entry("nested properties field", "properties.metrics.cpu", `jsonb_path_query_first(properties, '$.metrics.cpu') as "metrics_cpu"`),
		Entry("custom JSON column", "custom.field", `custom->>'field' as "field"`),
		Entry("nested custom JSON column", "custom.nested.value", `custom->'nested'->>'value' as "nested_value"`),
		Entry("non-JSON field", "name", "name"),
	)
})
