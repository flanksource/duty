package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
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

var _ = Describe("BuildSelectClause", func() {
	DescribeTable("should generate correct SELECT clause for various combinations",
		func(groupBy []string, aggregates []types.AggregationField, expected string) {
			result := query.BuildSelectClause(groupBy, aggregates)
			Expect(result).To(Equal(expected))
		},
		Entry("empty inputs",
			[]string{},
			[]types.AggregationField{},
			""),
		Entry("simple GROUP BY only",
			[]string{"name", "type"},
			[]types.AggregationField{},
			"name, type"),
		Entry("JSON GROUP BY fields",
			[]string{"labels.env", "config.version"},
			[]types.AggregationField{},
			`labels->>'env' as "env", config->>'version' as "version"`),
		Entry("mixed GROUP BY fields",
			[]string{"name", "labels.env"},
			[]types.AggregationField{},
			`name, labels->>'env' as "env"`),
		Entry("nested JSON GROUP BY field",
			[]string{"config.author.name"},
			[]types.AggregationField{},
			`config->'author'->>'name' as "author_name"`),
		Entry("simple COUNT aggregation",
			[]string{},
			[]types.AggregationField{{Function: "COUNT", Field: "*", Alias: "total"}},
			"COUNT(*) AS total"),
		Entry("simple SUM aggregation",
			[]string{},
			[]types.AggregationField{{Function: "SUM", Field: "size", Alias: "total_size"}},
			"SUM(size) AS total_size"),
		Entry("JSON field COUNT aggregation",
			[]string{},
			[]types.AggregationField{{Function: "COUNT", Field: "labels.env", Alias: "env_count"}},
			`COUNT(labels->>'env') AS env_count`),
		Entry("JSON field AVG aggregation (numeric)",
			[]string{},
			[]types.AggregationField{{Function: "AVG", Field: "config.cpu", Alias: "avg_cpu"}},
			`AVG(CAST(config->>'cpu' AS NUMERIC)) AS avg_cpu`),
		Entry("JSON field MAX aggregation (numeric)",
			[]string{},
			[]types.AggregationField{{Function: "MAX", Field: "config.memory", Alias: "max_memory"}},
			`MAX(CAST(config->>'memory' AS NUMERIC)) AS max_memory`),
		Entry("JSON field MIN aggregation (numeric)",
			[]string{},
			[]types.AggregationField{{Function: "MIN", Field: "config.disk", Alias: "min_disk"}},
			`MIN(CAST(config->>'disk' AS NUMERIC)) AS min_disk`),
		Entry("nested JSON field numeric aggregation",
			[]string{},
			[]types.AggregationField{{Function: "SUM", Field: "config.resources.memory", Alias: "total_memory"}},
			`SUM(CAST(config->'resources'->>'memory' AS NUMERIC)) AS total_memory`),
		Entry("multiple aggregations",
			[]string{},
			[]types.AggregationField{
				{Function: "COUNT", Field: "*", Alias: "total"},
				{Function: "SUM", Field: "size", Alias: "total_size"},
				{Function: "AVG", Field: "config.cpu", Alias: "avg_cpu"},
			},
			`COUNT(*) AS total, SUM(size) AS total_size, AVG(CAST(config->>'cpu' AS NUMERIC)) AS avg_cpu`),
	)
})
