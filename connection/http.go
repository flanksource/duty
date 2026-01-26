package connection

import (
	"encoding/json"
	"fmt"
	netHTTP "net/http"
	"strings"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/labstack/echo/v4"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
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
	ConnectionName      string `json:"connection,omitempty" yaml:"connection,omitempty"`
	types.HTTPBasicAuth `json:",inline"`
	URL                 string         `json:"url,omitempty" yaml:"url,omitempty"`
	Bearer              types.EnvVar   `json:"bearer,omitempty" yaml:"bearer,omitempty"`
	OAuth               types.OAuth    `json:"oauth,omitempty" yaml:"oauth,omitempty"`
	TLS                 TLSConfig      `json:"tls,omitempty" yaml:"tls,omitempty"`
	Headers             []types.EnvVar `json:"headers,omitempty" yaml:"headers,omitempty"`
}

func (t *HTTPConnection) FromModel(connection models.Connection) error {
	t.URL = connection.URL
	t.TLS.InsecureSkipVerify = connection.InsecureTLS

	if err := t.HTTPBasicAuth.Username.Scan(connection.Username); err != nil {
		return fmt.Errorf("error scanning username: %w", err)
	}
	if err := t.HTTPBasicAuth.Password.Scan(connection.Password); err != nil {
		return fmt.Errorf("error scanning password: %w", err)
	}

	if bearer := connection.Properties["bearer"]; bearer != "" {
		if err := t.Bearer.Scan(bearer); err != nil {
			return fmt.Errorf("error scanning bearer: %w", err)
		}
	}

	if clientID := connection.Properties["clientID"]; clientID != "" {
		if err := t.OAuth.ClientID.Scan(clientID); err != nil {
			return fmt.Errorf("error scanning oauth client_id: %w", err)
		}
	}
	if clientSecret := connection.Properties["clientSecret"]; clientSecret != "" {
		if err := t.OAuth.ClientSecret.Scan(clientSecret); err != nil {
			return fmt.Errorf("error scanning oauth client_secret: %w", err)
		}
	}
	if tokenURL := connection.Properties["tokenURL"]; tokenURL != "" {
		t.OAuth.TokenURL = tokenURL
	}
	if params := connection.Properties["params"]; params != "" {
		if err := json.Unmarshal([]byte(params), &t.OAuth.Params); err != nil {
			return fmt.Errorf("error unmarshaling oauth params: %w", err)
		}
	}
	if scopes := connection.Properties["scopes"]; scopes != "" {
		t.OAuth.Scopes = strings.Split(scopes, ",")
	}

	if headers := connection.Properties["headers"]; headers != "" {
		if err := json.Unmarshal([]byte(headers), &t.Headers); err != nil {
			return fmt.Errorf("error unmarshaling headers: %w", err)
		}
	}

	return nil
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

	for i := range h.Headers {
		h.Headers[i].ValueStatic, err = ctx.GetEnvValueFromCache(h.Headers[i], namespace)
		if err != nil {
			return h, err
		}
	}

	return h, nil
}

func (h HTTPConnection) Transport() netHTTP.RoundTripper {
	rt := &httpConnectionRoundTripper{
		HTTPConnection: h,
		Base:           &netHTTP.Transport{},
	}
	return rt
}

type httpConnectionRoundTripper struct {
	HTTPConnection
	Base netHTTP.RoundTripper
}

func (rt *httpConnectionRoundTripper) RoundTrip(req *netHTTP.Request) (*netHTTP.Response, error) {
	conn := rt.HTTPConnection
	if !conn.HTTPBasicAuth.IsEmpty() {
		req.SetBasicAuth(conn.HTTPBasicAuth.GetUsername(), conn.HTTPBasicAuth.GetPassword())
	} else if !conn.Bearer.IsEmpty() {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+conn.Bearer.ValueStatic)
	} else if !conn.OAuth.IsEmpty() {
		oauthTransport := middlewares.NewOauthTransport(middlewares.OauthConfig{
			ClientID:     conn.OAuth.ClientID.String(),
			ClientSecret: conn.OAuth.ClientSecret.String(),
			TokenURL:     conn.OAuth.TokenURL,
			Params:       conn.OAuth.Params,
			Scopes:       conn.OAuth.Scopes,
		})
		rt.Base = oauthTransport.RoundTripper(rt.Base)
	}

	for _, header := range conn.Headers {
		if !header.IsEmpty() {
			req.Header.Set(header.Name, header.ValueStatic)
		}
	}

	if !conn.TLS.IsEmpty() {
		rt.TLS = conn.TLS
	}

	return rt.Base.RoundTrip(req)
}

// CreateHTTPClient requires a hydrated connection
func CreateHTTPClient(ctx ConnectionContext, conn HTTPConnection) (*http.Client, error) {
	client := http.NewClient()
	if !conn.HTTPBasicAuth.IsEmpty() {
		client.Auth(conn.GetUsername(), conn.GetPassword())
		client.Digest(conn.Digest)
		client.NTLM(conn.NTLM)
		client.NTLMV2(conn.NTLMV2)
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

	for _, header := range conn.Headers {
		if !header.IsEmpty() {
			client.Header(header.Name, header.ValueStatic)
		}
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
	case models.ConnectionTypeHTTP, models.ConnectionTypePrometheus:
		if err := httpConn.FromModel(conn); err != nil {
			return httpConn, err
		}

		if _, err := httpConn.Hydrate(ctx, conn.Namespace); err != nil {
			return httpConn, fmt.Errorf("error hydrating connection: %w", err)
		}

	default:
		return httpConn, fmt.Errorf("invalid connection type: %s", conn.Type)
	}

	return httpConn, nil
}
