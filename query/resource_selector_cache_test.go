package query

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestGetParsedResourceSelectorPEGUsesCache(t *testing.T) {
	g := gomega.NewWithT(t)

	resourceSelectorPEGCache.Flush()

	first, err := getParsedResourceSelectorPEG(`name="coredns",type="Kubernetes::Pod"`)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	second, err := getParsedResourceSelectorPEG(`name="coredns",type="Kubernetes::Pod"`)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(second.queryField).To(gomega.Equal(first.queryField))
	g.Expect(second.flatFields).To(gomega.Equal(first.flatFields))
	g.Expect(resourceSelectorPEGCache.ItemCount()).To(gomega.Equal(1))
}

func TestGetParsedResourceSelectorPEGCachesByPEGValue(t *testing.T) {
	g := gomega.NewWithT(t)

	resourceSelectorPEGCache.Flush()

	_, err := getParsedResourceSelectorPEG(`name="coredns"`)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = getParsedResourceSelectorPEG(`name="metrics-server"`)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(resourceSelectorPEGCache.ItemCount()).To(gomega.Equal(2))
}

func TestGetSelectorRequirementsUsesCache(t *testing.T) {
	g := gomega.NewWithT(t)

	resourceSelectorLabelRequirementsCache.Flush()

	first, err := getSelectorRequirements("cluster=aws")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	second, err := getSelectorRequirements("cluster=aws")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(second).To(gomega.Equal(first))
	g.Expect(resourceSelectorLabelRequirementsCache.ItemCount()).To(gomega.Equal(1))
}

func TestGetSelectorRequirementsReturnsErrorForInvalidSelector(t *testing.T) {
	g := gomega.NewWithT(t)

	resourceSelectorLabelRequirementsCache.Flush()

	_, err := getSelectorRequirements("=aws")
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resourceSelectorLabelRequirementsCache.ItemCount()).To(gomega.Equal(0))
}
