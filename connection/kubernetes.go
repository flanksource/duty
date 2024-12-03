package connection

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

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

	return nil, nil, nil
}

// +kubebuilder:object:generate=true
type KubernetesConnection struct {
	KubeconfigConnection `json:",inline"`

	EKS  *EKSConnection  `json:"eks,omitempty"`
	GKE  *GKEConnection  `json:"gke,omitempty"`
	CNRM *CNRMConnection `json:"cnrm,omitempty"`
}

func (t KubernetesConnection) ToModel() models.Connection {
	return models.Connection{
		Type:        models.ConnectionTypeKubernetes,
		Certificate: t.Kubeconfig.ValueStatic,
	}
}

func (t *KubernetesConnection) Populate(ctx context.Context) (kubernetes.Interface, *rest.Config, error) {
	if clientset, restConfig, err := t.KubeconfigConnection.Populate(ctx); err != nil {
		return nil, nil, nil
	} else if clientset != nil {
		return clientset, restConfig, nil
	}

	if t.GKE != nil {
		if err := t.GKE.Populate(ctx); err != nil {
			return nil, nil, err
		}

		return t.GKE.KubernetesClient(ctx)
	}

	if t.EKS != nil {
		if err := t.EKS.Populate(ctx); err != nil {
			return nil, nil, err
		}

		return t.EKS.KubernetesClient(ctx)
	}

	if t.CNRM != nil {
		if err := t.CNRM.Populate(ctx); err != nil {
			return nil, nil, err
		}

		return t.CNRM.KubernetesClient(ctx)
	}

	return nil, nil, nil
}
