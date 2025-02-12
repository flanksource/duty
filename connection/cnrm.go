package connection

import (
	"encoding/base64"
	"fmt"

	"github.com/flanksource/duty/context"
	dutyKube "github.com/flanksource/duty/kubernetes"
	container "google.golang.org/api/container/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// +kubebuilder:object:generate=true
type CNRMConnection struct {
	GKE GKEConnection `json:"gke" yaml:"gke"`

	ClusterResource          string `json:"clusterResource"`
	ClusterResourceNamespace string `json:"clusterResourceNamespace"`
}

func (t *CNRMConnection) Populate(ctx ConnectionContext) error {
	return t.GKE.Populate(ctx)
}

func (t *CNRMConnection) KubernetesClient(ctx context.Context) (kubernetes.Interface, *rest.Config, error) {
	cnrmCluster, restConfig, err := t.GKE.KubernetesClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client for GKE: %w", err)
	}

	containerResourceKubeClient, err := dutyKube.NewKubeClient(cnrmCluster, restConfig).
		GetClientByGroupVersionKind("container.cnrm.cloud.google.com", "v1beta1", "ContainerCluster")
	if err != nil {
		return nil, nil, err
	}

	obj, err := containerResourceKubeClient.Namespace(t.ClusterResourceNamespace).Get(ctx, t.ClusterResource, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	clusterResourceRestConfig, err := t.createRestConfigForClusteResource(ctx, obj)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(clusterResourceRestConfig)
	if err != nil {
		return nil, nil, err
	}

	return clientset, restConfig, nil
}

func (t *CNRMConnection) createRestConfigForClusteResource(ctx context.Context, clusterObj *unstructured.Unstructured) (*rest.Config, error) {
	endpoint, found, err := unstructured.NestedString(clusterObj.Object, "status", "endpoint")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to extract cluster endpoint from cluster resource: %w", err)
	}

	caCertB64, found, err := unstructured.NestedString(clusterObj.Object, "spec", "masterAuth", "clusterCaCertificate")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to extract cluster CA certificate from cluster resource: %w", err)
	}

	ca, err := base64.URLEncoding.DecodeString(caCertB64)
	if err != nil {
		return nil, fmt.Errorf("unable to decode cluster CA certificate: %w", err)
	}

	token, err := t.GKE.Token(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for gke: %w", err)
	}

	return &rest.Config{
		Host: endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
		BearerToken: token.AccessToken,
	}, nil
}
