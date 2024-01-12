package tests

import (
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
			components, err := duty.FindComponents(DefaultContext, td.Selectors, duty.PickColumns("id", "path"))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(components)).To(Equal(td.Results))
		})
	}
})
