package connection

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type GCPConnection struct {
	// ConnectionName of the connection. It'll be used to populate the endpoint and credentials.
	ConnectionName string        `yaml:"connection,omitempty" json:"connection,omitempty"`
	Endpoint       string        `yaml:"endpoint" json:"endpoint,omitempty"`
	Credentials    *types.EnvVar `yaml:"credentials" json:"credentials,omitempty"`

	// Skip TLS verify
	SkipTLSVerify bool `yaml:"skipTLSVerify,omitempty" json:"skipTLSVerify,omitempty"`
}

func (g *GCPConnection) Validate() *GCPConnection {
	if g == nil {
		return &GCPConnection{}
	}

	return g
}

func (g *GCPConnection) Token(ctx context.Context, scopes ...string) (*oauth2.Token, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(g.Credentials.ValueStatic), scopes...)
	if err != nil {
		return nil, err
	}

	tokenSource := creds.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return token, nil
}

// HydrateConnection attempts to find the connection by name
// and populate the endpoint and credentials.
func (g *GCPConnection) HydrateConnection(ctx ConnectionContext) error {
	connection, err := ctx.HydrateConnectionByURL(g.ConnectionName)
	if err != nil {
		return err
	}

	if connection != nil {
		g.Credentials = &types.EnvVar{ValueStatic: connection.Certificate}
		g.Endpoint = connection.URL
	}

	return nil
}

func (t *GCPConnection) GetCertificate() types.EnvVar {
	return utils.Deref(t.Credentials)
}

func (t *GCPConnection) GetURL() types.EnvVar {
	return types.EnvVar{ValueStatic: t.Endpoint}
}
