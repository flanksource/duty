package kubernetes

import (
	"testing"

	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseAPIVersionKind(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedGroup   string
		expectedVersion string
		expectedKind    string
		expectError     bool
	}{
		{
			name:            "core v1 resource",
			input:           "v1/Pod",
			expectedGroup:   "",
			expectedVersion: "v1",
			expectedKind:    "Pod",
			expectError:     false,
		},
		{
			name:            "apps group resource",
			input:           "apps/v1/Deployment",
			expectedGroup:   "apps",
			expectedVersion: "v1",
			expectedKind:    "Deployment",
			expectError:     false,
		},
		{
			name:            "domain-based group resource",
			input:           "serving.knative.dev/v1/Service",
			expectedGroup:   "serving.knative.dev",
			expectedVersion: "v1",
			expectedKind:    "Service",
			expectError:     false,
		},
		{
			name:            "custom resource definition",
			input:           "cert-manager.io/v1/Certificate",
			expectedGroup:   "cert-manager.io",
			expectedVersion: "v1",
			expectedKind:    "Certificate",
			expectError:     false,
		},
		{
			name:        "invalid format - no slash",
			input:       "Pod",
			expectError: true,
		},
		{
			name:        "invalid format - too many slashes",
			input:       "serving.knative.dev/v1/Service/extra",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			gvk, err := ParseAPIVersionKind(tt.input)
			if tt.expectError {
				g.Expect(err).To(gomega.HaveOccurred())
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(gvk.Group).To(gomega.Equal(tt.expectedGroup))
				g.Expect(gvk.Version).To(gomega.Equal(tt.expectedVersion))
				g.Expect(gvk.Kind).To(gomega.Equal(tt.expectedKind))
			}
		})
	}
}

func TestParseAPIVersionKindReturnsGVK(t *testing.T) {
	g := gomega.NewWithT(t)

	gvk, err := ParseAPIVersionKind("apps/v1/Deployment")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(gvk).To(gomega.Equal(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}))
}
