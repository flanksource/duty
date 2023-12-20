package connection

import (
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type GCSConnection struct {
	GCPConnection `json:",inline"`
	Bucket        string `yaml:"bucket,omitempty" json:"bucket,omitempty"`
}

func (g *GCSConnection) Validate() *GCSConnection {
	if g == nil {
		return &GCSConnection{}
	}

	return g
}

// HydrateConnection attempts to find the connection by name
// and populate the endpoint and credentials.
func (g *GCSConnection) HydrateConnection(ctx ConnectionContext) error {
	connection, err := ctx.HydrateConnectionByURL(g.ConnectionName)
	if err != nil {
		return err
	}

	if connection != nil {
		g.Credentials = &types.EnvVar{ValueStatic: connection.Certificate}
		g.Endpoint = connection.URL
		if val, ok := connection.Properties["bucket"]; ok {
			g.Bucket = val
		}
	}

	return nil
}

func (t *GCSConnection) GetProperties() map[string]string {
	return map[string]string{
		"bucket": t.Bucket,
	}
}
