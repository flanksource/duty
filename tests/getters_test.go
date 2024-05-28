package tests

import (
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("FindChecks", func() {
	type testRecord struct {
		Name      string
		Selectors []types.ResourceSelector
		Results   int
	}

	testData := []testRecord{
		{
			Name:      "empty",
			Selectors: []types.ResourceSelector{},
			Results:   0,
		},
		{
			Name:      "name",
			Selectors: []types.ResourceSelector{{Name: dummy.LogisticsAPIHealthHTTPCheck.Name}},
			Results:   1,
		},
		{
			Name:      "names",
			Selectors: []types.ResourceSelector{{Name: dummy.LogisticsAPIHealthHTTPCheck.Name}, {Name: dummy.LogisticsAPIHomeHTTPCheck.Name}, {Name: dummy.LogisticsDBCheck.Name}},
			Results:   3,
		},
		{
			Name:      "names but different namespace",
			Selectors: []types.ResourceSelector{{Namespace: "kube-system", Name: dummy.LogisticsAPIHealthHTTPCheck.Name}, {Namespace: "kube-system", Name: dummy.LogisticsAPIHomeHTTPCheck.Name}},
			Results:   0,
		},
		{
			Name:      "types",
			Selectors: []types.ResourceSelector{{Types: []string{dummy.LogisticsDBCheck.Type}}},
			Results:   1,
		},
		{
			Name:      "repeated (types) to test cache",
			Selectors: []types.ResourceSelector{{Types: []string{dummy.LogisticsDBCheck.Type}}},
			Results:   1,
		},
		{
			Name:      "agentID",
			Selectors: []types.ResourceSelector{{Agent: dummy.CartAPIHeathCheckAgent.AgentID.String()}},
			Results:   1,
		},
		{
			Name:      "type & statuses",
			Selectors: []types.ResourceSelector{{Types: []string{dummy.LogisticsDBCheck.Type}, Statuses: []string{string(dummy.LogisticsDBCheck.Status)}}},
			Results:   1,
		},
		{
			Name:      "label selector",
			Selectors: []types.ResourceSelector{{LabelSelector: "app=logistics"}},
			Results:   3,
		},
	}

	for i := range testData {
		td := testData[i]

		ginkgo.It(td.Name, func() {
			components, err := query.FindCheckIDs(DefaultContext, td.Selectors...)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(components)).To(Equal(td.Results))
		})
	}
})

var _ = ginkgo.Describe("FindConfigs", func() {
	type testRecord struct {
		Name      string
		Selectors []types.ResourceSelector
		Results   int
	}

	testData := []testRecord{
		{
			Name:      "empty",
			Selectors: []types.ResourceSelector{},
			Results:   0,
		},
		{
			Name:      "name",
			Selectors: []types.ResourceSelector{{Name: lo.FromPtr(dummy.KubernetesNodeA.Name)}},
			Results:   1,
		},
		{
			Name:      "name but different namespace",
			Selectors: []types.ResourceSelector{{Namespace: "kube-system", Name: lo.FromPtr(dummy.KubernetesNodeA.Name)}},
			Results:   0,
		},
		{
			Name:      "types",
			Selectors: []types.ResourceSelector{{Types: []string{lo.FromPtr(dummy.KubernetesNodeA.Type)}}},
			Results:   3,
		},
		{
			Name:      "repeated (types) to test cache",
			Selectors: []types.ResourceSelector{{Types: []string{lo.FromPtr(dummy.KubernetesNodeA.Type)}}},
			Results:   3,
		},
		{
			Name:      "label selector",
			Selectors: []types.ResourceSelector{{LabelSelector: "role=worker"}},
			Results:   2,
		},
		{
			Name:      "field selector",
			Selectors: []types.ResourceSelector{{FieldSelector: "config_class=Deployment"}},
			Results:   3,
		},
	}

	for i := range testData {
		td := testData[i]

		ginkgo.It(td.Name, func() {
			components, err := query.FindConfigIDsByResourceSelector(DefaultContext, td.Selectors...)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(components)).To(Equal(td.Results))
		})
	}
})

var _ = ginkgo.Describe("FindComponent", func() {
	type testRecord struct {
		Name      string
		Selectors []types.ResourceSelector
		Results   int
	}

	testData := []testRecord{
		{
			Name:      "name",
			Selectors: []types.ResourceSelector{{Name: dummy.Logistics.Name}},
			Results:   1,
		},
		{
			Name:      "names",
			Selectors: []types.ResourceSelector{{Name: dummy.Logistics.Name}, {Name: dummy.LogisticsAPI.Name}},
			Results:   2,
		},
		{
			Name:      "names but different namespace",
			Selectors: []types.ResourceSelector{{Namespace: "kube-system", Name: dummy.Logistics.Name}, {Namespace: "kube-system", Name: dummy.LogisticsAPI.Name}},
			Results:   0,
		},
		{
			Name:      "types",
			Selectors: []types.ResourceSelector{{Types: []string{dummy.Logistics.Type}}},
			Results:   1,
		},
		{
			Name:      "repeated (types) to test cache",
			Selectors: []types.ResourceSelector{{Types: []string{dummy.Logistics.Type}}},
			Results:   1,
		},
		{
			Name:      "agentID",
			Selectors: []types.ResourceSelector{{Agent: dummy.PaymentsAPI.AgentID.String()}},
			Results:   1,
		},
		{
			Name:      "type & statuses",
			Selectors: []types.ResourceSelector{{Types: []string{"KubernetesPod"}, Statuses: []string{string(types.ComponentStatusHealthy)}}},
			Results:   3,
		},
		{
			Name:      "empty",
			Selectors: []types.ResourceSelector{},
			Results:   0,
		},
		{
			Name:      "name and label selector that have overlaps",
			Selectors: []types.ResourceSelector{{Name: dummy.Logistics.Name, LabelSelector: "telemetry=enabled"}},
			Results:   1,
		},
		{
			Name:      "field selector",
			Selectors: []types.ResourceSelector{{FieldSelector: "name=kustomize"}},
			Results:   1,
		},
	}

	for i := range testData {
		td := testData[i]

		ginkgo.It(td.Name, func() {
			components, err := query.FindComponentIDs(DefaultContext, td.Selectors...)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(components)).To(Equal(td.Results))
		})
	}
})
