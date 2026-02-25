package connection

import (
	gocontext "context"
	"testing"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/onsi/gomega"
)

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
