package tests

import (
	gocontext "context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var _ = Describe("HTTP Connection", func() {
	It("should send requests with auth, headers, and payload", func() {
		expectedPayload := `{"message":"hello"}`
		testPath := "/api"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()

			Expect(r.Method).To(Equal(http.MethodPost))
			Expect(r.URL.Path).To(Equal(testPath))

			username, password, ok := r.BasicAuth()
			Expect(ok).To(BeTrue())
			Expect(username).To(Equal("test-user"))
			Expect(password).To(Equal("test-password"))

			Expect(r.Header.Get("X-API-Key")).To(Equal("secret"))
			Expect(r.Header.Get("X-Custom-Header")).To(Equal("custom-value"))

			body, err := io.ReadAll(r.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal(expectedPayload))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		headersJSON, err := json.Marshal([]types.EnvVar{
			{
				Name: "X-API-Key",
				ValueFrom: &types.EnvVarSource{
					SecretKeyRef: &types.SecretKeySelector{
						LocalObjectReference: types.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "foo",
					},
				},
			},
			{
				Name:        "X-Custom-Header",
				ValueStatic: "custom-value",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		conn := models.Connection{
			Name:      "http-test",
			Namespace: "default",
			Type:      models.ConnectionTypeHTTP,
			URL:       server.URL + testPath,
			Username:  "test-user",
			Password:  "test-password",
			Source:    models.SourceUI,
			Properties: map[string]string{
				"headers": string(headersJSON),
			},
		}

		Expect(DefaultContext.DB().Create(&conn).Error).ToNot(HaveOccurred())
		defer DefaultContext.DB().Delete(&conn)

		storedConn, err := DefaultContext.GetConnection("http-test", "default")
		Expect(err).ToNot(HaveOccurred())
		Expect(storedConn).ToNot(BeNil())

		httpConn, err := connection.NewHTTPConnection(DefaultContext, *storedConn)
		Expect(err).ToNot(HaveOccurred())

		client, err := connection.CreateHTTPClient(DefaultContext, httpConn)
		Expect(err).ToNot(HaveOccurred())

		resp, err := client.R(gocontext.Background()).
			Header("Content-Type", "application/json").
			Post(httpConn.URL, expectedPayload)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})
