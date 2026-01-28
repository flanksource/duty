package connection

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/flanksource/duty/context"
	"golang.org/x/oauth2/google"
	container "google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// +kubebuilder:object:generate=true
type GKEConnection struct {
	GCPConnection `json:",inline" yaml:",inline"`

	ProjectID string `json:"projectID"`
	Zone      string `json:"zone"`
	Cluster   string `json:"cluster"`
}

func (t *GKEConnection) Populate(ctx ConnectionContext) error {
	return t.GCPConnection.HydrateConnection(ctx)
}

func (t *GKEConnection) Validate() *GKEConnection {
	if t == nil {
		return &GKEConnection{}
	}

	return t
}

func (t *GKEConnection) Client(ctx context.Context) (*container.Service, error) {
	t = t.Validate()

	var clientOpts []option.ClientOption

	if t.Endpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(t.Endpoint))
	}

	if t.SkipTLSVerify {
		insecureHTTPClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		clientOpts = append(clientOpts, option.WithHTTPClient(insecureHTTPClient))
	}

	if t.Credentials != nil && !t.Credentials.IsEmpty() {
		credential, err := ctx.GetEnvValueFromCache(*t.Credentials, ctx.GetNamespace())
		if err != nil {
			return nil, err
		}
		creds, err := google.CredentialsFromJSON(ctx, []byte(credential), container.CloudPlatformScope)
		if err != nil {
			return nil, err
		}
		clientOpts = append(clientOpts, option.WithCredentials(creds))
	} else {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	}

	svc, err := container.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (t *GKEConnection) KubernetesClient(ctx context.Context, freshToken bool) (kubernetes.Interface, *rest.Config, error) {
	containerService, err := t.Client(ctx)
	if err != nil {
		return nil, nil, err
	}

	clusterName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", t.ProjectID, t.Zone, t.Cluster)
	cluster, err := containerService.Projects.Locations.Clusters.Get(clusterName).Context(ctx).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	token, err := t.GCPConnection.Token(ctx, freshToken, container.CloudPlatformScope)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get token for gke: %w", err)
	}

	ca, err := base64.URLEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decode cluster CA certificate: %w", err)
	}

	restConfig := &rest.Config{
		Host:        "https://" + cluster.Endpoint,
		BearerToken: token.AccessToken,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	return clientset, restConfig, nil
}
