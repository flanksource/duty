package kubernetes

import (
	"errors"
	"fmt"

	"github.com/flanksource/commons/hash"
	"k8s.io/client-go/rest"
	clientcmdAPI "k8s.io/client-go/tools/clientcmd/api/v1"

	// NOTE: must use sigs.k8s.io/yaml instead of gopkg.in/yaml.v2 because it uses json struct tags
	// gopkg.in/yaml will not pickup "current-context"
	"sigs.k8s.io/yaml"
)

// RestConfigFingerprint generates a unique SHA-256 hash to identify the Kubernetes API server
// and client authentication details from the REST configuration.
func RestConfigFingerprint(rc *rest.Config) string {
	if rc == nil {
		return ""
	}

	return hash.Sha256Hex(fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s",
		rc.Host,
		rc.APIPath,
		rc.Username,
		rc.Password,
		rc.BearerToken,
		rc.BearerTokenFile,
		rc.TLSClientConfig.CertData))
}

func GetAPIServer(kubeconfigRaw []byte) (string, error) {
	var kubeconfig clientcmdAPI.Config
	if err := yaml.Unmarshal(kubeconfigRaw, &kubeconfig); err != nil {
		return "", err
	}

	var currentCluster string
	for _, c := range kubeconfig.Contexts {
		if c.Name == kubeconfig.CurrentContext {
			currentCluster = c.Context.Cluster
			break
		}
	}

	for _, c := range kubeconfig.Clusters {
		if c.Name == currentCluster {
			return c.Cluster.Server, nil
		}
	}

	return "", errors.New("current cluster not found")
}
