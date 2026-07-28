// Tests the rendering of field path keys into jsonpath expressions
// used to address values inside jsonb columns.
package query

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("toJSONPath", func() {
	DescribeTable("renders keys as a jsonpath expression",
		func(segments []string, expected string) {
			Expect(toJSONPath(segments)).To(Equal(expected))
		},
		Entry("simple key", []string{"account"}, `$.account`),
		Entry("nested keys", []string{"metadata", "name"}, `$.metadata.name`),
		Entry("array index", []string{"containers[0]", "name"}, `$.containers[0].name`),
		Entry("key with a dash", []string{"mission-control"}, `$."mission-control"`),
		Entry("key with a slash", []string{"topic/mission-control"}, `$."topic/mission-control"`),
		Entry("key with dots", []string{"app.kubernetes.io/name"}, `$."app.kubernetes.io/name"`),
		Entry("key starting with a digit", []string{"8s"}, `$."8s"`),
		Entry("key with a quote", []string{`he"llo`}, `$."he\"llo"`),
		Entry("key with a backslash", []string{`he\llo`}, `$."he\\llo"`),
	)
})
