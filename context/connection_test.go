package context

import (
	"context"

	commons "github.com/flanksource/commons/context"
	"github.com/flanksource/duty/models"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Connection Tests", func() {
	ginkgo.Describe("GetConnectionNameType", func() {
		testCases := []struct {
			name       string
			connection string
			Expect     struct {
				name      string
				namespace string
				found     bool
			}
		}{
			{
				name:       "valid connection string",
				connection: "connection://default/mission_control",
				Expect: struct {
					name      string
					namespace string
					found     bool
				}{
					name:      "mission_control",
					namespace: "default",
					found:     true,
				},
			},
			{
				name:       "empty namespace",
				connection: "connection://  /mission_control",
				Expect: struct {
					name      string
					namespace string
					found     bool
				}{
					name:      "mission_control",
					namespace: "",
					found:     true,
				},
			},
			{
				name:       "invalid connection string",
				connection: "invalid-connection-string",
				Expect: struct {
					name      string
					namespace string
					found     bool
				}{
					name:      "",
					namespace: "",
					found:     false,
				},
			},
			{
				name:       "empty connection string",
				connection: "",
				Expect: struct {
					name      string
					namespace string
					found     bool
				}{
					name:      "",
					namespace: "",
					found:     false,
				},
			},
			{
				name:       "namespace only",
				connection: "connection://default/",
				Expect: struct {
					name      string
					namespace string
					found     bool
				}{
					name:      "",
					namespace: "default",
					found:     false,
				},
			},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			ginkgo.Context(tc.name, func() {
				ginkgo.It("should return the correct name, namespace, and found status", func() {
					name, namespace, found := extractConnectionNameType(tc.connection)
					Expect(name).To(Equal(tc.Expect.name))
					Expect(namespace).To(Equal(tc.Expect.namespace))
					Expect(found).To(Equal(tc.Expect.found))
				})
			})
		}
	})

	ginkgo.Describe("HydrateConnection", func() {
		dummyContext := Context{
			Context: commons.NewContext(context.Background()),
		}

		testCases := []struct {
			name       string
			connection models.Connection
			expect     string
		}{
			{
				name: "properties templating",
				connection: models.Connection{
					URL:      "postgres://$(username):$(password)@$(properties.host):$(properties.port)/$(properties.database)",
					Username: "the-username",
					Password: "the-password",
					Properties: map[string]string{
						"host":     "localhost",
						"database": "mission_control",
						"port":     "5443",
					},
				},
				expect: "postgres://the-username:the-password@localhost:5443/mission_control",
			},
			{
				name: "space and newline trimming",
				connection: models.Connection{
					URL: `

                        postgres://$(username):$(password)@$(properties.host):$(properties.port)/$(properties.database)

                    `,
					Username: "  the-username",
					Password: "the-password  ",
					Properties: map[string]string{
						"host":     "localhost",
						"database": "mission_control",
						"port":     "5443",
					},
				},
				expect: "postgres://the-username:the-password@localhost:5443/mission_control",
			},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			ginkgo.Context(tc.name, func() {
				ginkgo.It("should return the correct hydrated URL", func() {
					resp, err := HydrateConnection(dummyContext, &tc.connection)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.URL).To(Equal(tc.expect))
				})
			})
		}
	})
})
