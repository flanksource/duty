package kubernetes

import (
	"encoding/json"
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

func TestNewClientFromPathOrConfigAcceptsJSONStringKubeconfig(t *testing.T) {
	g := gomega.NewWithT(t)

	kubeconfig, err := os.ReadFile(filepath.Join("testdata", "kubeconfig.yaml"))
	g.Expect(err).To(gomega.BeNil())

	encoded, err := json.Marshal(string(kubeconfig))
	g.Expect(err).To(gomega.BeNil())

	client, restConfig, err := NewClientFromPathOrConfigWithMiddleware(nil, string(encoded), nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(client).ToNot(gomega.BeNil())
	g.Expect(restConfig).ToNot(gomega.BeNil())
	g.Expect(restConfig.Host).To(gomega.Equal("https://10.99.99.222:6443"))
}
