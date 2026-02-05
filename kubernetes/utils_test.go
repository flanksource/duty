package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestGetAPIServer(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfigPath string
		expected       string
	}{
		{
			name:           "valid kubeconfig",
			kubeconfigPath: "kubeconfig.yaml",
			expected:       "https://10.99.99.222:6443",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			f, err := os.ReadFile(filepath.Join("testdata", tc.kubeconfigPath))
			g.Expect(err).To(gomega.BeNil())

			result, err := GetAPIServer(f)
			g.Expect(err).To(gomega.BeNil())
			g.Expect(result).To(gomega.Equal(tc.expected))
		})
	}
}
