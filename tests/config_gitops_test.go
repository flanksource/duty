package tests

import (
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/gomplate/v3"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var gitopsPath = "aws-demo/spec/namespaces/flux/namespace.yaml"

var gitopsFixtures = []struct {
	id       any
	expected string
}{
	{dummy.Namespace, gitopsPath},
	{dummy.Namespace.ID, gitopsPath},
	{dummy.Namespace.ID.String(), gitopsPath},
	{dummy.Namespace.AsMap(), gitopsPath},
}
var _ = ginkgo.Describe("Config Gitops Source", ginkgo.Ordered, func() {
	ginkgo.It("should resolve kustomize references", func() {
		Expect(dummy.Kustomization.ID.String()).NotTo(BeEmpty())
		Expect(dummy.GitRepository.ID.String()).NotTo(BeEmpty())
		Expect(dummy.Namespace.ID.String()).NotTo(BeEmpty())

		// Config Traverse uses the 7d summary view internally
		err := job.RefreshConfigItemSummary7d(DefaultContext)
		Expect(err).To(BeNil())

		source, err := query.GetGitOpsSource(DefaultContext, dummy.Namespace.ID)
		Expect(err).To(BeNil())
		Expect(source.Kustomize.Path).To(Equal("./aws-demo/spec"))
		Expect(source.Git.File).To(Equal("aws-demo/spec/namespaces/flux/namespace.yaml"))
		Expect(source.Git.Dir).To(Equal("aws-demo/spec/namespaces/flux"))
		Expect(source.Git.URL).To(Equal("ssh://git@github.com/flanksource/sandbox.git"))
		Expect(source.Git.Link).To(Equal("https://github.com/flanksource/sandbox/tree/main/aws-demo/spec/namespaces/flux/namespace.yaml"))
		Expect(source.Git.Branch).To(Equal("main"))
	})

	ginkgo.It("should resolve references using CEL", func() {
		for _, fixture := range gitopsFixtures {
			out, err := DefaultContext.RunTemplate(gomplate.Template{
				Expression: "gitops.source(id).git.file",
			}, map[string]any{
				"id": fixture.id,
			})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(fixture.expected))

		}
	})

	ginkgo.It("should resolve references using gomplate", func() {
		for _, fixture := range gitopsFixtures {
			out, err := DefaultContext.RunTemplate(gomplate.Template{
				Template: "{{ ( .id | gitops_source ).git.file }}",
			}, map[string]any{
				"id": fixture.id,
			})
			Expect(err).To(BeNil())
			Expect(out).To(Equal(fixture.expected))
		}
	})
})
