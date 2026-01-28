package connection

import (
	"crypto/tls"
	"net/http"

	gcs "cloud.google.com/go/storage"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
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

func (g *GCSConnection) Client(ctx context.Context) (*gcs.Client, error) {
	g = g.Validate()
	var client *gcs.Client
	var err error

	var clientOpts []option.ClientOption

	if g.Endpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(g.Endpoint))
	}

	if g.SkipTLSVerify {
		insecureHTTPClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		clientOpts = append(clientOpts, option.WithHTTPClient(insecureHTTPClient))
	}

	if g.Credentials != nil && !g.Credentials.IsEmpty() {
		credential, err := ctx.GetEnvValueFromCache(*g.Credentials, ctx.GetNamespace())
		if err != nil {
			return nil, err
		}
		creds, err := google.CredentialsFromJSON(ctx, []byte(credential), gcs.ScopeReadWrite)
		if err != nil {
			return nil, err
		}
		clientOpts = append(clientOpts, option.WithCredentials(creds))
	} else {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	}

	client, err = gcs.NewClient(ctx.Context, clientOpts...)
	if err != nil {
		return nil, err
	}

	return client, nil
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
