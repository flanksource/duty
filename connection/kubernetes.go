package connection

import (
	"fmt"

	"github.com/flanksource/duty/context"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// +kubebuilder:object:generate=true
type KubeconfigConnection struct {
	// Connection name to populate kubeconfig
	ConnectionName string        `json:"connection,omitempty"`
	Kubeconfig     *types.EnvVar `json:"kubeconfig,omitempty"`
}

func (t KubeconfigConnection) Populate(ctx context.Context) (kubernetes.Interface, *rest.Config, error) {
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
}

// String returns a human readable string representation of the KubernetesConnection
func (c KubernetesConnection) String() string {
	if c.ConnectionName != "" {
		return "connection://" + c.ConnectionName
	}
	if c.Kubeconfig != nil {
		if c.Kubeconfig.ValueStatic != "" {
			config, err := clientcmd.Load([]byte(c.Kubeconfig.ValueStatic))
			if err == nil {
				if !lo.Contains([]string{"", "default", "gke_default", "kubernetes"}, config.CurrentContext) {
					// context name is usually more descriptive
					return "kubeconfig://" + config.CurrentContext
				}
				for k := range config.Clusters {
					return "kubeconfig://" + k
				}
			}
		}
		return "kubeconfig://" + c.Kubeconfig.String()
	}
	if c.EKS != nil {
		return fmt.Sprintf("eks://%s", c.EKS.Cluster)
	}
	if c.GKE != nil {
		return fmt.Sprintf("gke://%s", c.GKE.Cluster)
	}
	if c.CNRM != nil {
		return fmt.Sprintf("cnrm://%s", c.CNRM.ClusterResource)
	}
	return "local://"

}

// Hash returns a unique identifier of a KubernetesConnection, suitable for caching
func (c KubernetesConnection) Hash() string {
	if c.ConnectionName != "" {
		return "connection=" + c.ConnectionName
	}
	if c.Kubeconfig != nil {
		return "kubeconfig=" + c.Kubeconfig.String()
	}
	if c.EKS != nil {
		return fmt.Sprintf("eks=%s/%v", c.EKS.Cluster, c.EKS.ToModel())
	}
	if c.GKE != nil {
		return fmt.Sprintf("gke=%s/%v", c.GKE.Cluster, c.GKE.ToModel())
	}
	if c.CNRM != nil {
		return fmt.Sprintf("cnrm=%s/%v", c.CNRM.ClusterResource, c.GKE.ToModel())
	}
	return "local"
}

func (c KubernetesConnection) CanExpire() bool {
	return c.EKS != nil ||
		c.GKE != nil ||
		c.CNRM != nil ||
		lo.FromPtr(c.Kubeconfig).ValueFrom != nil
}

func (t KubernetesConnection) ToModel() models.Connection {
	return models.Connection{
		Type:        models.ConnectionTypeKubernetes,
		Certificate: t.Kubeconfig.ValueStatic,
	}
}

func (t KubernetesConnection) Populate(ctx context.Context, freshToken bool) (kubernetes.Interface, *rest.Config, error) {
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
