package connection

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/labstack/echo/v4"
)

// +kubebuilder:object:generate=true
type TLSConfig struct {
	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	// HandshakeTimeout defaults to 10 seconds
	HandshakeTimeout time.Duration `json:"handshakeTimeout,omitempty" yaml:"handshakeTimeout,omitempty"`
	// PEM encoded certificate of the CA to verify the server certificate
	CA types.EnvVar `json:"ca,omitempty" yaml:"ca,omitempty"`
	// PEM encoded client certificate
	Cert types.EnvVar `json:"cert,omitempty" yaml:"cert,omitempty"`
	// PEM encoded client private key
	Key types.EnvVar `json:"key,omitempty" yaml:"key,omitempty"`
}

func (t TLSConfig) IsEmpty() bool {
	return t.CA.IsEmpty() || t.Cert.IsEmpty() || t.Key.IsEmpty()
}

// +kubebuilder:object:generate=true
type HTTPConnection struct {
	ConnectionName       string `json:"connection,omitempty" yaml:"connection,omitempty"`
	types.Authentication `json:",inline"`
	URL                  string       `json:"url,omitempty" yaml:"url,omitempty"`
	Bearer               types.EnvVar `json:"bearer,omitempty" yaml:"bearer,omitempty"`
	OAuth                types.OAuth  `json:"oauth,omitempty" yaml:"oauth,omitempty"`
	TLS                  TLSConfig    `json:"tls,omitempty" yaml:"tls,omitempty"`
}

func (h HTTPConnection) GetEndpoint() string {
	return h.URL
}

func (h *HTTPConnection) Hydrate(ctx ConnectionContext, namespace string) (*HTTPConnection, error) {
	var err error
	if h.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(h.ConnectionName)
		if err != nil {
			return h, fmt.Errorf("could not hydrate connection[%s]: %w", h.ConnectionName, err)
		}
		if connection == nil {
			return h, fmt.Errorf("connection[%s] not found", h.ConnectionName)
		}
		*h, err = NewHTTPConnection(ctx, *connection)
		if err != nil {
			return h, fmt.Errorf("error creating connection from model: %w", err)
		}
	}

	// URL can be an EnvVar string so we
	// typecase to EnvVar and scan it first
	var url types.EnvVar
	if err := url.Scan(h.URL); err != nil {
		return h, err
	}
	h.URL, err = ctx.GetEnvValueFromCache(url, namespace)
	if err != nil {
		return h, err
	}

	h.Authentication.Username.ValueStatic, err = ctx.GetEnvValueFromCache(h.Authentication.Username, namespace)
	if err != nil {
		return h, err
	}
	h.Authentication.Password.ValueStatic, err = ctx.GetEnvValueFromCache(h.Authentication.Password, namespace)
	if err != nil {
		return h, err
	}

	h.Bearer.ValueStatic, err = ctx.GetEnvValueFromCache(h.Bearer, namespace)
	if err != nil {
		return h, err
	}

	h.OAuth.ClientID.ValueStatic, err = ctx.GetEnvValueFromCache(h.OAuth.ClientID, namespace)
	if err != nil {
		return h, err
	}
	h.OAuth.ClientSecret.ValueStatic, err = ctx.GetEnvValueFromCache(h.OAuth.ClientSecret, namespace)
	if err != nil {
		return h, err
	}

	h.TLS.Key.ValueStatic, err = ctx.GetEnvValueFromCache(h.TLS.Key, namespace)
	if err != nil {
		return h, err
	}
	h.TLS.CA.ValueStatic, err = ctx.GetEnvValueFromCache(h.TLS.CA, namespace)
	if err != nil {
		return h, err
	}
	h.TLS.Cert.ValueStatic, err = ctx.GetEnvValueFromCache(h.TLS.Cert, namespace)
	if err != nil {
		return h, err
	}
	return h, nil
}

// CreateHTTPClient requires a hydrated connection
func CreateHTTPClient(ctx ConnectionContext, conn HTTPConnection) (*http.Client, error) {
	client := http.NewClient()
	if !conn.Authentication.IsEmpty() {
		client.Auth(conn.GetUsername(), conn.GetPassword())
	} else if !conn.Bearer.IsEmpty() {
		client.Header(echo.HeaderAuthorization, "Bearer "+conn.Bearer.ValueStatic)
	} else if !conn.OAuth.IsEmpty() {
		client.OAuth(middlewares.OauthConfig{
			ClientID:     conn.OAuth.ClientID.ValueStatic,
			ClientSecret: conn.OAuth.ClientSecret.ValueStatic,
			TokenURL:     conn.OAuth.TokenURL,
			Params:       conn.OAuth.Params,
			Scopes:       conn.OAuth.Scopes,
		})
	}

	if !conn.TLS.IsEmpty() {
		_, err := client.TLSConfig(http.TLSConfig{
			CA:                 conn.TLS.CA.ValueStatic,
			Cert:               conn.TLS.Cert.ValueStatic,
			Key:                conn.TLS.Key.ValueStatic,
			InsecureSkipVerify: conn.TLS.InsecureSkipVerify,
			HandshakeTimeout:   conn.TLS.HandshakeTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("error setting tls config: %w", err)
		}
	}

	return client, nil
}

func NewHTTPConnection(ctx ConnectionContext, conn models.Connection) (HTTPConnection, error) {
	var httpConn HTTPConnection
	switch conn.Type {
	case models.ConnectionTypeHTTP:
		if err := httpConn.Username.Scan(conn.Username); err != nil {
			return httpConn, fmt.Errorf("error scanning username: %w", err)
		}
		if err := httpConn.Password.Scan(conn.Password); err != nil {
			return httpConn, fmt.Errorf("error scanning password: %w", err)
		}

		if bearer := conn.Properties["bearer"]; bearer != "" {
			if err := httpConn.Bearer.Scan(bearer); err != nil {
				return httpConn, fmt.Errorf("error scanning bearer: %w", err)
			}
		}

		if oauthClientID := conn.Properties["clientID"]; oauthClientID != "" {
			if err := httpConn.OAuth.ClientID.Scan(oauthClientID); err != nil {
				return httpConn, fmt.Errorf("error scanning oauth_client_id: %w", err)
			}
		}
		if oauthClientSecret := conn.Properties["clientSecret"]; oauthClientSecret != "" {
			if err := httpConn.OAuth.ClientSecret.Scan(oauthClientSecret); err != nil {
				return httpConn, fmt.Errorf("error scanning oauth_client_secret: %w", err)
			}
		}
		if oauthTokenURL := conn.Properties["tokenURL"]; oauthTokenURL != "" {
			httpConn.OAuth.TokenURL = oauthTokenURL
		}
		if oauthParams := conn.Properties["params"]; oauthParams != "" {
			if err := json.Unmarshal([]byte(oauthParams), &httpConn.OAuth.Params); err != nil {
				return httpConn, fmt.Errorf("error unmarshaling params:%s in oauth: %w", oauthParams, err)
			}
		}
		if oauthScopes := conn.Properties["scopes"]; oauthScopes != "" {
			httpConn.OAuth.Scopes = strings.Split(oauthScopes, ",")
		}

		if _, err := httpConn.Hydrate(ctx, conn.Namespace); err != nil {
			return httpConn, fmt.Errorf("error hydrating connection: %w", err)
		}

	default:
		return httpConn, fmt.Errorf("invalid connection type: %s", conn.Type)
	}

	return httpConn, nil
}
