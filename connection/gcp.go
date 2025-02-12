package connection

import (
	"crypto/tls"
	"net/http"

	"fmt"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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

func (t *GCPConnection) ToModel() models.Connection {
	return models.Connection{
		Name:        t.ConnectionName,
		URL:         t.Endpoint,
		Certificate: t.Credentials.String(),
		InsecureTLS: t.SkipTLSVerify,
	}
}

func (t *GCPConnection) FromModel(connection models.Connection) {
	t.ConnectionName = connection.Name
	t.Credentials = &types.EnvVar{ValueStatic: connection.Certificate}
	t.Endpoint = connection.URL
	t.SkipTLSVerify = connection.InsecureTLS
}

func (g *GCPConnection) TokenSource(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(g.Credentials.ValueStatic), scopes...)
	if err != nil {
		return nil, err
	}

	tokenSource := creds.TokenSource
	return tokenSource, nil
}

func (conn *GCPConnection) Client(ctx context.Context) (*gcs.Client, error) {
	conn = conn.Validate()
	var client *gcs.Client
	var err error

	var clientOpts []option.ClientOption

	if conn.Endpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(conn.Endpoint))
	}

	if conn.SkipTLSVerify {
		insecureHTTPClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		clientOpts = append(clientOpts, option.WithHTTPClient(insecureHTTPClient))
	}

	if conn.Credentials != nil && !conn.Credentials.IsEmpty() {
		credential, err := ctx.GetEnvValueFromCache(*conn.Credentials, ctx.GetNamespace())
		if err != nil {
			return nil, err
		}
		clientOpts = append(clientOpts, option.WithCredentialsJSON([]byte(credential)))
	} else {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	}

	client, err = gcs.NewClient(ctx.Context, clientOpts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (g *GCPConnection) Validate() *GCPConnection {
	if g == nil {
		return &GCPConnection{}
	}

	return g
}

func (g *GCPConnection) Token(ctx context.Context, scopes ...string) (*oauth2.Token, error) {
	cacheKey := tokenCacheKey("gcp", hash.Sha256Hex(g.Credentials.ValueStatic), strings.Join(scopes, ","))
	if found, ok := tokenCache.Get(cacheKey); ok {
		return found.(*oauth2.Token), nil
	}

	creds, err := google.CredentialsFromJSON(ctx, []byte(g.Credentials.ValueStatic), scopes...)
	if err != nil {
		return nil, err
	}

	tokenSource := creds.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	tokenCache.Set(cacheKey, token, time.Until(token.Expiry))
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

	if g.Credentials != nil {
		if cred, err := ctx.GetEnvValueFromCache(*g.Credentials, ctx.GetNamespace()); err != nil {
			return fmt.Errorf("could get gcloud credentials from env var: %w", err)
		} else {
			g.Credentials.ValueStatic = cred
		}
	}

	return nil
}

func (t *GCPConnection) GetCertificate() types.EnvVar {
	return utils.Deref(t.Credentials)
}

func (t *GCPConnection) GetURL() types.EnvVar {
	return types.EnvVar{ValueStatic: t.Endpoint}
}
