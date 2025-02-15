package kubernetes

import (
	"fmt"

	"github.com/flanksource/commons/hash"
	"k8s.io/client-go/rest"
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
