package connection

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flanksource/duty/cache"
	"github.com/flanksource/duty/context"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type KubeconfigConnection struct {
	// Connection name to populate kubeconfig
	ConnectionName string        `json:"connection,omitempty"`
	Kubeconfig     *types.EnvVar `json:"kubeconfig,omitempty"`
}

func (t *KubeconfigConnection) Populate(ctx context.Context) (kubernetes.Interface, *rest.Config, error) {
	if t.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(t.ConnectionName)
		if err != nil {
			return nil, nil, err
		} else if connection == nil {
			return nil, nil, fmt.Errorf("connection[%s] not found", t.ConnectionName)
		}

		t.Kubeconfig.ValueStatic = connection.Certificate
	}

	if t.Kubeconfig != nil {
		if v, err := ctx.GetEnvValueFromCache(*t.Kubeconfig, ctx.GetNamespace()); err != nil {
			return nil, nil, err
		} else {
			t.Kubeconfig.ValueStatic = v
		}

		return dutyKubernetes.NewClientFromPathOrConfig(ctx.Logger, t.Kubeconfig.ValueStatic)
	}

	return dutyKubernetes.NewClient(ctx.Logger)
}

// +kubebuilder:object:generate=true
type KubernetesConnection struct {
	KubeconfigConnection `json:",inline"`

	EKS  *EKSConnection  `json:"eks,omitempty"`
	GKE  *GKEConnection  `json:"gke,omitempty"`
	CNRM *CNRMConnection `json:"cnrm,omitempty"`

	Client *dutyKubernetes.Client
}

func (t KubernetesConnection) ToModel() models.Connection {
	return models.Connection{
		Type:        models.ConnectionTypeKubernetes,
		Certificate: t.Kubeconfig.ValueStatic,
	}
}

var k8sClientCache = cache.NewCache[*dutyKubernetes.Client]("k8s-client-cache", 24*time.Hour)

func (t *KubernetesConnection) Populate(ctx context.Context, freshToken bool) (*dutyKubernetes.Client, error) {
	clientSet, restConfig, err := t.populate(ctx, freshToken)
	if err != nil {
		return nil, fmt.Errorf("error populating kubernetes connection: %w", err)
	}

	cacheKey := dutyKubernetes.RestConfigFingerprint(restConfig)
	if c, err := k8sClientCache.Get(ctx, cacheKey); err == nil {
		return c, nil
	}

	c := dutyKubernetes.NewKubeClient(clientSet, restConfig)
	k8sClientCache.Set(ctx, cacheKey, c)
	return c, nil
}

func (t *KubernetesConnection) populate(ctx context.Context, freshToken bool) (kubernetes.Interface, *rest.Config, error) {
	if clientset, restConfig, err := t.KubeconfigConnection.Populate(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to populate kube config connection: %w", err)
	} else if clientset != nil {
		return clientset, restConfig, nil
	}

	if t.GKE != nil {
		if err := t.GKE.Populate(ctx); err != nil {
			return nil, nil, err
		}

		return t.GKE.KubernetesClient(ctx, freshToken)
	}

	if t.EKS != nil {
		if err := t.EKS.Populate(ctx); err != nil {
			return nil, nil, err
		}

		return t.EKS.KubernetesClient(ctx, freshToken)
	}

	if t.CNRM != nil {
		if err := t.CNRM.Populate(ctx); err != nil {
			return nil, nil, err
		}

		return t.CNRM.KubernetesClient(ctx, freshToken)
	}

	return nil, nil, nil
}
