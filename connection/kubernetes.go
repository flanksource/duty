package connection

import (
	"fmt"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type KubernetesConnection struct {
	ConnectionName string       `json:"connection,omitempty"`
	KubeConfig     types.EnvVar `json:"kubeconfig,omitempty"`
}

func (t KubernetesConnection) ToModel() models.Connection {
	return models.Connection{
		Type:        models.ConnectionTypeKubernetes,
		Certificate: t.KubeConfig.ValueStatic,
	}
}

func (t *KubernetesConnection) Populate(ctx ConnectionContext) error {
	if t.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(t.ConnectionName)
		if err != nil {
			return err
		} else if connection == nil {
			return fmt.Errorf("connection[%s] not found", t.ConnectionName)
		}

		t.KubeConfig.ValueStatic = connection.Certificate
	}

	if v, err := ctx.GetEnvValueFromCache(t.KubeConfig, ctx.GetNamespace()); err != nil {
		return err
	} else {
		t.KubeConfig.ValueStatic = v
	}

	return nil
}
