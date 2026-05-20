package connection

import (
	gocontext "context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/onsi/gomega"
)

func TestTLSConfigIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		config  TLSConfig
		expects bool
	}{
		{name: "empty", config: TLSConfig{}, expects: true},
		{name: "insecure skip verify", config: TLSConfig{InsecureSkipVerify: true}},
		{name: "handshake timeout", config: TLSConfig{HandshakeTimeout: time.Second}},
		{name: "ca only", config: TLSConfig{CA: types.EnvVar{ValueStatic: "ca"}}},
		{name: "cert only", config: TLSConfig{Cert: types.EnvVar{ValueStatic: "cert"}}},
		{name: "key only", config: TLSConfig{Key: types.EnvVar{ValueStatic: "key"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tc.config.IsEmpty()).To(gomega.Equal(tc.expects))
		})
	}
}

func TestHTTPConnectionTransportTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})

	tests := []struct {
		name       string
		config     TLSConfig
		expectsErr bool
	}{
		{
			name:       "default TLS rejects unknown CA",
			expectsErr: true,
		},
		{
			name:   "custom CA",
			config: TLSConfig{CA: types.EnvVar{ValueStatic: string(certPEM)}},
		},
		{
			name:   "insecure skip verify",
			config: TLSConfig{InsecureSkipVerify: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			rt, err := HTTPConnection{TLS: tc.config}.Transport()
			g.Expect(err).ToNot(gomega.HaveOccurred())
			client := &http.Client{Transport: rt}

			resp, err := client.Get(server.URL)
			if tc.expectsErr {
				g.Expect(err).To(gomega.HaveOccurred())
				return
			}

			g.Expect(err).ToNot(gomega.HaveOccurred())
			defer resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
		})
	}
}

func TestHTTPConnectionTransportMTLS(t *testing.T) {
	g := gomega.NewWithT(t)
	clientCertPEM, clientKeyPEM, clientCAPool := newClientCertificate(t)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCAPool,
	}
	server.StartTLS()
	defer server.Close()

	serverCAPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	rt, err := HTTPConnection{TLS: TLSConfig{
		CA:   types.EnvVar{ValueStatic: string(serverCAPEM)},
		Cert: types.EnvVar{ValueStatic: string(clientCertPEM)},
		Key:  types.EnvVar{ValueStatic: string(clientKeyPEM)},
	}}.Transport()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	client := &http.Client{Transport: rt}
	resp, err := client.Get(server.URL)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	defer resp.Body.Close()
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}

func newClientCertificate(t *testing.T) (certPEM, keyPEM []byte, caPool *x509.CertPool) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create ca certificate: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca certificate: %v", err)
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create client certificate: %v", err)
	}

	caPool = x509.NewCertPool()
	caPool.AddCert(caCert)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})
	return certPEM, keyPEM, caPool
}

func TestCreateHTTPClientWithTLS(t *testing.T) {
	g := gomega.NewWithT(t)
	client, err := CreateHTTPClient(nil, HTTPConnection{TLS: TLSConfig{InsecureSkipVerify: true}})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(client).ToNot(gomega.BeNil())
}

func TestHTTPConnectionPretty(t *testing.T) {
	tests := []struct {
		name     string
		conn     HTTPConnection
		contains []string
		excludes []string
	}{
		{
			name: "basic URL with connection name",
			conn: HTTPConnection{
				ConnectionName: "my-api",
				URL:            "https://api.example.com/v1/users?status=active&limit=10",
			},
			contains: []string{
				"my-api",
				"api.example.com",
				"/v1/users",
				"status=active",
				"limit=10",
			},
		},
		{
			name: "basic auth",
			conn: HTTPConnection{
				URL: "https://example.com",
				HTTPBasicAuth: types.HTTPBasicAuth{
					Authentication: types.Authentication{
						Username: types.EnvVar{ValueStatic: "admin"},
						Password: types.EnvVar{ValueStatic: "secret"},
					},
				},
			},
			contains: []string{"Basic", "admin", "****"},
			excludes: []string{"secret"},
		},
		{
			name: "bearer token",
			conn: HTTPConnection{
				URL:    "https://example.com",
				Bearer: types.EnvVar{ValueStatic: "tok123"},
			},
			contains: []string{"Bearer", "****"},
			excludes: []string{"tok123"},
		},
		{
			name: "oauth",
			conn: HTTPConnection{
				URL: "https://graph.microsoft.com/v1.0/groups",
				OAuth: types.OAuth{
					ClientID:     types.EnvVar{ValueStatic: "abc123"},
					ClientSecret: types.EnvVar{ValueStatic: "secret"},
					TokenURL:     "https://login.microsoftonline.com/tenant/oauth2/v2.0/token",
				},
			},
			contains: []string{"OAuth", "abc123"},
			excludes: []string{"secret"},
		},
		{
			name: "aws sigv4",
			conn: HTTPConnection{
				URL: "https://es.amazonaws.com",
				AWSSigV4: &AWSSigV4{
					Service: "es",
				},
			},
			contains: []string{"AWS SigV4", "es"},
		},
		{
			name: "insecure TLS",
			conn: HTTPConnection{
				URL: "https://example.com",
				TLS: TLSConfig{InsecureSkipVerify: true},
			},
			contains: []string{"insecure TLS"},
		},
		{
			name: "headers",
			conn: HTTPConnection{
				URL: "https://example.com",
				Headers: []types.EnvVar{
					{Name: "Content-Type", ValueStatic: "application/json"},
					{Name: "Accept", ValueStatic: "text/plain"},
				},
			},
			contains: []string{"Content-Type", "Accept"},
		},
		{
			name: "URL without query params",
			conn: HTTPConnection{
				URL: "https://example.com/api/health",
			},
			contains: []string{"example.com", "/api/health"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			result := tc.conn.Pretty().String()
			for _, want := range tc.contains {
				g.Expect(result).To(gomega.ContainSubstring(want))
			}
			for _, exclude := range tc.excludes {
				g.Expect(result).ToNot(gomega.ContainSubstring(exclude))
			}
		})
	}
}

type mockConnectionContext struct {
	gocontext.Context
	connection *models.Connection
}

func (m mockConnectionContext) HydrateConnectionByURL(string) (*models.Connection, error) {
	return m.connection, nil
}

func (m mockConnectionContext) GetEnvValueFromCache(env types.EnvVar, _ string) (string, error) {
	return env.ValueStatic, nil
}

func (m mockConnectionContext) GetNamespace() string { return "default" }

func TestHydratePreservesInlineOAuthScopes(t *testing.T) {
	azureConn := &models.Connection{
		Type:     models.ConnectionTypeAzure,
		URL:      "https://graph.microsoft.com",
		Username: "client-id",
		Password: "client-secret",
		Properties: types.JSONStringMap{
			"tenant": "my-tenant",
		},
	}

	tests := []struct {
		name           string
		inlineScopes   []string
		expectedScopes []string
	}{
		{
			name:           "inline scopes override connection defaults",
			inlineScopes:   []string{"Group.Read.All", "User.Read.All"},
			expectedScopes: []string{"Group.Read.All", "User.Read.All"},
		},
		{
			name:           "connection defaults used when no inline scopes",
			inlineScopes:   nil,
			expectedScopes: []string{"https://graph.microsoft.com/.default"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ctx := mockConnectionContext{
				Context:    gocontext.Background(),
				connection: azureConn,
			}

			conn := HTTPConnection{
				ConnectionName: "connection://azure",
				OAuth:          types.OAuth{Scopes: tc.inlineScopes},
			}

			result, err := conn.Hydrate(ctx, "default")
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(result.OAuth.Scopes).To(gomega.HaveLen(len(tc.expectedScopes)))
			for i, scope := range result.OAuth.Scopes {
				g.Expect(scope).To(gomega.Equal(tc.expectedScopes[i]))
			}
		})
	}
}
