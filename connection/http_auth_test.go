package connection

import (
	gocontext "context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	"github.com/flanksource/duty/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP authentication")
}

var _ = Describe("HTTP authentication", func() {
	DescribeTable("rejects removed NTLM authentication before constructing a client or transport", func(ntlm, ntlmV2 bool) {
		conn := HTTPConnection{HTTPBasicAuth: types.HTTPBasicAuth{NTLM: ntlm, NTLMV2: ntlmV2}}
		_, err := conn.Transport()
		Expect(err).To(MatchError("NTLM authentication is no longer supported"))
		_, err = CreateHTTPClient(nil, conn)
		Expect(err).To(MatchError("NTLM authentication is no longer supported"))
	}, Entry("NTLM", true, false), Entry("NTLMv2", false, true), Entry("both", true, true))

	DescribeTable("signs requests with SDK credentials and their session token", func(useCommons bool) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Seen-Authorization", r.Header.Get("Authorization"))
			w.Header().Set("Seen-Token", r.Header.Get("X-Amz-Security-Token"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		conn := HTTPConnection{
			AWSSigV4: &AWSSigV4{Service: "execute-api"},
			AwsConfig: &cachedAWSConfig{aws.Config{
				Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("example", "example-secret", "example-session"),
			}},
		}
		response := requestHTTPConnection(conn, server.URL, useCommons)
		defer response.Body.Close()
		Expect(response.Header.Get("Seen-Authorization")).To(And(
			HavePrefix("AWS4-HMAC-SHA256 Credential=example/"), ContainSubstring("/us-east-1/execute-api/aws4_request"),
		))
		Expect(response.Header.Get("Seen-Token")).To(Equal("example-session"))
	}, Entry("commons client", true), Entry("standard transport", false))

	DescribeTable("exchanges OAuth credentials and sends the bearer token", func(useCommons bool) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"access_token":"example-token","token_type":"Bearer","expires_in":3600}`))
				if err != nil {
					panic(err)
				}
				return
			}
			w.Header().Set("Seen-Authorization", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		conn := HTTPConnection{OAuth: types.OAuth{
			ClientID: types.EnvVar{ValueStatic: "example-client"}, ClientSecret: types.EnvVar{ValueStatic: "example-secret"},
			TokenURL: server.URL + "/token",
		}}
		response := requestHTTPConnection(conn, server.URL, useCommons)
		defer response.Body.Close()
		Expect(response.Header.Get("Seen-Authorization")).To(Equal("Bearer example-token"))
	}, Entry("commons client", true), Entry("standard transport", false))

	DescribeTable("preserves SDK credential expiration semantics", func(canExpire bool) {
		expires := time.Now().Add(time.Hour).UTC()
		cfg := cachedAWSConfig{aws.Config{Region: "us-east-1", Credentials: aws.CredentialsProviderFunc(func(gocontext.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "example", SecretAccessKey: "example-secret", SessionToken: "example-session", CanExpire: canExpire, Expires: expires}, nil
		})}}
		converted, err := cfg.signingConfig().Credentials.Retrieve(gocontext.Background())
		Expect(err).NotTo(HaveOccurred())
		expected := awssigv4.Credentials{AccessKeyID: "example", SecretAccessKey: "example-secret", SessionToken: "example-session"}
		if canExpire {
			expected.Expires = expires
		}
		Expect(converted).To(Equal(expected))
	}, Entry("temporary credentials", true), Entry("permanent credentials", false))

	It("passes request cancellation and retrieval errors through the SDK adapter", func() {
		cfg := cachedAWSConfig{aws.Config{Credentials: aws.CredentialsProviderFunc(func(ctx gocontext.Context) (aws.Credentials, error) {
			return aws.Credentials{}, ctx.Err()
		})}}
		ctx, cancel := gocontext.WithCancel(gocontext.Background())
		cancel()
		_, err := cfg.signingConfig().Credentials.Retrieve(ctx)
		Expect(err).To(MatchError(gocontext.Canceled))
	})
})

func requestHTTPConnection(conn HTTPConnection, url string, useCommons bool) *http.Response {
	GinkgoHelper()
	if useCommons {
		client, err := CreateHTTPClient(nil, conn)
		Expect(err).NotTo(HaveOccurred())
		response, err := client.R(gocontext.Background()).Get(url)
		Expect(err).NotTo(HaveOccurred())
		return response.Response
	}
	transport, err := conn.Transport()
	Expect(err).NotTo(HaveOccurred())
	response, err := (&http.Client{Transport: transport}).Get(url)
	Expect(err).NotTo(HaveOccurred())
	return response
}
