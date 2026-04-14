package types

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestResourceSelectorHasWildcard(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(resourceSelectorHasWildcard(ResourceSelector{
		Name:          "api-server",
		TagSelector:   "cluster=aws",
		LabelSelector: "app=backend",
		FieldSelector: "owner=platform",
		Types:         Items{"Kubernetes::Pod"},
		Statuses:      Items{"healthy"},
		Health:        "healthy",
	})).To(gomega.BeFalse())

	g.Expect(resourceSelectorHasWildcard(ResourceSelector{
		TagSelector: "cluster=*",
	})).To(gomega.BeTrue())
}
