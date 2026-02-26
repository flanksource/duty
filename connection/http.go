package connection

import (
	"encoding/json"
	"fmt"
	netHTTP "net/http"
	netURL "net/url"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
	"github.com/labstack/echo/v4"

	"github.com/flanksource/duty/context"
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
type AWSSigV4 struct {
	AWSConnection `json:",inline" yaml:",inline"`
	Service       string `json:"service,omitempty" yaml:"service,omitempty"`
}

// cachedAWSConfig wraps aws.Config with DeepCopy methods so controller-gen
// can handle it without calling nonexistent methods on aws.Config.
type cachedAWSConfig struct{ aws.Config }

func (in *cachedAWSConfig) DeepCopyInto(out *cachedAWSConfig) { *out = *in }
func (in *cachedAWSConfig) DeepCopy() *cachedAWSConfig {
	if in == nil {
		return nil
	}
	out := new(cachedAWSConfig)
	*out = *in
	return out
}

// +kubebuilder:object:generate=true
type HTTPConnection struct {
	ConnectionName      string `json:"connection,omitempty" yaml:"connection,omitempty"`
	types.HTTPBasicAuth `json:",inline"`
	URL                 string           `json:"url,omitempty" yaml:"url,omitempty"`
	Bearer              types.EnvVar     `json:"bearer,omitempty" yaml:"bearer,omitempty"`
	OAuth               types.OAuth      `json:"oauth,omitempty" yaml:"oauth,omitempty"`
	TLS                 TLSConfig        `json:"tls,omitempty" yaml:"tls,omitempty"`
	Headers             []types.EnvVar   `json:"headers,omitempty" yaml:"headers,omitempty"`
	AWSSigV4            *AWSSigV4        `json:"awsSigV4,omitempty" yaml:"awsSigV4,omitempty"`
	awsConfig           *cachedAWSConfig `json:"-"` // cached; populated during Hydrate
}

func (t HTTPConnection) Pretty() api.Text {
	s := clicky.Text("")

	if t.ConnectionName != "" {
		s = s.AddText("🔗 ", "text-blue-500").AddText(t.ConnectionName, "font-bold")
	}

	if t.URL != "" {
		s = s.NewLine()
		if parsed, err := netURL.Parse(t.URL); err == nil && parsed.Host != "" {
			s = s.AddText(parsed.Scheme+"://"+parsed.Host, "font-bold text-blue-600").
				AddText(parsed.Path, "text-gray-500")
			for i, key := range sortedQueryKeys(parsed.Query()) {
				prefix := "?"
				if i > 0 {
					prefix = "&"
				}
				s = s.NewLine().AddText(fmt.Sprintf("  %s%s=%s", prefix, key, parsed.Query().Get(key)), "text-gray-400")
			}
		} else {
			s = s.AddText(t.URL, "text-blue-600")
		}
	}

	if !t.HTTPBasicAuth.IsEmpty() {
		s = s.NewLine().AddText("🔑 Basic ", "text-yellow-600").AddText(t.GetUsername()+"/****", "text-gray-500")
	} else if !t.Bearer.IsEmpty() {
		s = s.NewLine().AddText("🔑 Bearer ****", "text-yellow-600")
	} else if !t.OAuth.IsEmpty() {
		s = s.NewLine().AddText("🔑 OAuth", "text-yellow-600")
		if !t.OAuth.ClientID.IsEmpty() {
			s = s.AddText(" client-id: ", "text-gray-500").AddText(t.OAuth.ClientID.ValueStatic, "text-gray-400")
		}
	}

	if t.AWSSigV4 != nil {
		s = s.NewLine().AddText("🔑 AWS SigV4", "text-yellow-600")
		if t.AWSSigV4.Service != "" {
			s = s.AddText(" "+t.AWSSigV4.Service, "text-gray-500")
		}
	}

	if t.TLS.InsecureSkipVerify {
		s = s.NewLine().AddText("⚠ insecure TLS", "text-red-500")
	}

	if len(t.Headers) > 0 {
		var names []string
		for _, h := range t.Headers {
			if h.Name != "" {
				names = append(names, h.Name)
			}
		}
		if len(names) > 0 {
			s = s.NewLine().AddText("Headers: "+strings.Join(names, ", "), "text-gray-500")
		}
	}

	return s
}

func sortedQueryKeys(values netURL.Values) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// mergeHeaders merges connection-level and inline (action-level) headers.
// Inline headers override connection headers with the same name.
func mergeHeaders(conn, inline []types.EnvVar) []types.EnvVar {
	overridden := make(map[string]struct{}, len(inline))
	for _, h := range inline {
		overridden[h.Name] = struct{}{}
	}
	merged := make([]types.EnvVar, 0, len(conn)+len(inline))
	for _, h := range conn {
		if _, ok := overridden[h.Name]; !ok {
			merged = append(merged, h)
		}
	}
	return append(merged, inline...)
}

func (t HTTPConnection) String() string {
	return t.Pretty().String()
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

	if awsSigV4 := connection.Properties["awsSigV4"]; awsSigV4 != "" {
		t.AWSSigV4 = &AWSSigV4{}
		if err := json.Unmarshal([]byte(awsSigV4), t.AWSSigV4); err != nil {
			return fmt.Errorf("error unmarshaling awsSigV4: %w", err)
		}
	}

	return nil
}

func (h HTTPConnection) GetEndpoint() string {
	return h.URL
}

func (h *HTTPConnection) Hydrate(ctx ConnectionContext, namespace string) (*HTTPConnection, error) {

	logger.V(6).Infof("hydrating HTTP connection: %s", h.Pretty().ANSI())
	var err error
	if h.ConnectionName != "" {
		existing := *h
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
		if existing.URL != "" {
			h.URL = existing.URL
		}
		if !existing.HTTPBasicAuth.IsEmpty() {
			h.HTTPBasicAuth = existing.HTTPBasicAuth
		}
		if !existing.Bearer.IsEmpty() {
			h.Bearer = existing.Bearer
		}
		if !existing.OAuth.ClientID.IsEmpty() {
			h.OAuth.ClientID = existing.OAuth.ClientID
		}
		if !existing.OAuth.ClientSecret.IsEmpty() {
			h.OAuth.ClientSecret = existing.OAuth.ClientSecret
		}
		if existing.OAuth.TokenURL != "" {
			h.OAuth.TokenURL = existing.OAuth.TokenURL
		}
		if len(existing.OAuth.Scopes) > 0 {
			h.OAuth.Scopes = existing.OAuth.Scopes
		}
		if len(existing.OAuth.Params) > 0 {
			h.OAuth.Params = existing.OAuth.Params
		}
		if len(existing.Headers) > 0 {
			h.Headers = mergeHeaders(h.Headers, existing.Headers)
		}
		if !existing.TLS.CA.IsEmpty() || !existing.TLS.Cert.IsEmpty() || !existing.TLS.Key.IsEmpty() || existing.TLS.InsecureSkipVerify || existing.TLS.HandshakeTimeout != 0 {
			h.TLS = existing.TLS
		}
		if existing.AWSSigV4 != nil {
			h.AWSSigV4 = existing.AWSSigV4
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

	if h.AWSSigV4 != nil {
		if err := h.AWSSigV4.Populate(ctx); err != nil {
			return h, fmt.Errorf("error populating aws sigv4 connection: %w", err)
		}
		if dutyCtx, ok := ctx.(context.Context); ok {
			cfg, err := h.AWSSigV4.Client(dutyCtx)
			if err != nil {
				return h, fmt.Errorf("error getting aws config for sigv4: %w", err)
			}
			h.awsConfig = &cachedAWSConfig{cfg}
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
			req.Header.Add(header.Name, header.ValueStatic)
		}
	}

	if !conn.TLS.IsEmpty() {
		rt.TLS = conn.TLS
	}

	if conn.AWSSigV4 != nil && conn.awsConfig != nil {
		rt.Base = middlewares.NewAWSSigv4Transport(middlewares.AWSSigv4Config{
			Region:              conn.awsConfig.Region,
			Service:             conn.AWSSigV4.Service,
			CredentialsProvider: conn.awsConfig.Credentials,
		}, rt.Base)
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

	if conn.AWSSigV4 != nil && conn.awsConfig != nil {
		client.AWSAuthSigV4(conn.awsConfig.Config)
		if conn.AWSSigV4.Service != "" {
			client.AWSService(conn.AWSSigV4.Service)
		}
	}

	// if properties.On(false, "http.log.curl") {
	// 	client.CurlLog()
	// }

	return client, nil
}

func NewHTTPConnection(ctx ConnectionContext, conn models.Connection) (HTTPConnection, error) {
	var httpConn HTTPConnection
	switch conn.Type {
	case models.ConnectionTypeHTTP, models.ConnectionTypePrometheus, "":
		if err := httpConn.FromModel(conn); err != nil {
			return httpConn, err
		}

	case models.ConnectionTypeAWS:
		httpConn.URL = conn.URL
		httpConn.TLS.InsecureSkipVerify = conn.InsecureTLS
		httpConn.AWSSigV4 = &AWSSigV4{}
		httpConn.AWSSigV4.FromModel(conn)

	case models.ConnectionTypeAzure:
		if err := httpConn.FromModel(conn); err != nil {
			return httpConn, err
		}

		httpConn.HTTPBasicAuth = types.HTTPBasicAuth{} // Azure connections should not use basic auth
		httpConn.URL = conn.URL
		httpConn.TLS.InsecureSkipVerify = conn.InsecureTLS
		var azure AzureConnection
		azure.FromModel(conn)
		if httpConn.OAuth.ClientID.IsEmpty() {
			httpConn.OAuth.ClientID = types.EnvVar{ValueStatic: azure.ClientID.String()}
		}
		if httpConn.OAuth.ClientSecret.IsEmpty() {
			httpConn.OAuth.ClientSecret = types.EnvVar{ValueStatic: azure.ClientSecret.String()}
		}
		if httpConn.OAuth.TokenURL == "" {
			httpConn.OAuth.TokenURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", azure.TenantID)
		}
		if httpConn.OAuth.Scopes == nil {
			httpConn.OAuth.Scopes = []string{"https://graph.microsoft.com/.default"}
		}

	default:
		return httpConn, fmt.Errorf("invalid connection type: %s", conn.Type)
	}

	if _, err := httpConn.Hydrate(ctx, conn.Namespace); err != nil {
		return httpConn, fmt.Errorf("error hydrating connection: %w", err)
	}

	return httpConn, nil
}
