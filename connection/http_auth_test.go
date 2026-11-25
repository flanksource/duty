package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	"github.com/flanksource/duty/types"
	"github.com/onsi/gomega"
)

func TestHTTPAuthenticationOAuth(t *testing.T) {
	for _, mode := range []string{"transport", "client"} {
		t.Run(mode, func(t *testing.T) {
			g := gomega.NewWithT(t)
			var tokenScope, tokenAudience, authorization string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					tokenScope = r.FormValue("scope")
					tokenAudience = r.FormValue("audience")
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"example-token","token_type":"Bearer","expires_in":3600}`))
					return
				}
				authorization = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			conn := HTTPConnection{OAuth: types.OAuth{
				ClientID: types.EnvVar{ValueStatic: "example-client"}, ClientSecret: types.EnvVar{ValueStatic: "example-secret"},
				TokenURL: server.URL + "/token", Scopes: []string{"read", "write"}, Params: map[string]string{"audience": "example-api"},
			}}
			response, err := sendAuthenticatedRequest(conn, mode, server.URL)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			defer response.Body.Close()
			g.Expect(response.StatusCode).To(gomega.Equal(http.StatusNoContent))
			g.Expect(authorization).To(gomega.Equal("Bearer example-token"))
			g.Expect(tokenScope).To(gomega.Equal("read write"))
			g.Expect(tokenAudience).To(gomega.Equal("example-api"))
		})
	}
}

func TestHTTPAuthenticationAWSSigning(t *testing.T) {
	for _, mode := range []string{"transport", "client"} {
		t.Run(mode, func(t *testing.T) {
			g := gomega.NewWithT(t)
			var authorization, sessionToken string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorization = r.Header.Get("Authorization")
				sessionToken = r.Header.Get("X-Amz-Security-Token")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			conn := HTTPConnection{
				AWSSigV4: &AWSSigV4{Service: "execute-api"},
				AwsConfig: &cachedAWSConfig{aws.Config{
					Region: "eu-west-1", Credentials: credentials.NewStaticCredentialsProvider("example-key", "example-secret", "example-session"),
				}},
			}
			response, err := sendAuthenticatedRequest(conn, mode, server.URL)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			defer response.Body.Close()
			g.Expect(response.StatusCode).To(gomega.Equal(http.StatusNoContent))
			g.Expect(authorization).To(gomega.HavePrefix("AWS4-HMAC-SHA256 Credential=example-key/"))
			g.Expect(authorization).To(gomega.ContainSubstring("/eu-west-1/execute-api/aws4_request"))
			g.Expect(sessionToken).To(gomega.Equal("example-session"))
		})
	}
}

func TestAWSSigningCredentials(t *testing.T) {
	g := gomega.NewWithT(t)
	type contextKey struct{}
	expires := time.Now().Add(time.Hour)
	expectedError := errors.New("credentials unavailable")
	for _, providerError := range []error{nil, expectedError} {
		cfg := awsSigningConfig(aws.Config{Region: "eu-west-1", Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			g.Expect(ctx.Value(contextKey{})).To(gomega.Equal("request-context"))
			return aws.Credentials{
				AccessKeyID: "example-key", SecretAccessKey: "example-secret", SessionToken: "example-session", CanExpire: true, Expires: expires,
			}, providerError
		})})
		actual, err := cfg.Credentials.Retrieve(context.WithValue(context.Background(), contextKey{}, "request-context"))
		if providerError != nil {
			g.Expect(err).To(gomega.MatchError(expectedError))
			continue
		}
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(cfg.Region).To(gomega.Equal("eu-west-1"))
		g.Expect(actual).To(gomega.Equal(awssigv4.Credentials{
			AccessKeyID: "example-key", SecretAccessKey: "example-secret", SessionToken: "example-session", Expires: expires,
		}))
	}
}

func TestHTTPAuthenticationRejectsNTLM(t *testing.T) {
	for _, auth := range []types.HTTPBasicAuth{{NTLM: true}, {NTLMV2: true}} {
		g := gomega.NewWithT(t)
		_, err := CreateHTTPClient(nil, HTTPConnection{HTTPBasicAuth: auth})
		g.Expect(err).To(gomega.MatchError("NTLM authentication is no longer supported"))
	}
}

func sendAuthenticatedRequest(conn HTTPConnection, mode, url string) (*http.Response, error) {
	if mode == "client" {
		client, err := CreateHTTPClient(nil, conn)
		if err != nil {
			return nil, err
		}
		response, err := client.R(context.Background()).Get(url)
		if err != nil {
			return nil, err
		}
		return response.Response, nil
	}
	transport, err := conn.Transport()
	if err != nil {
		return nil, err
	}
	return (&http.Client{Transport: transport}).Get(url)
}
