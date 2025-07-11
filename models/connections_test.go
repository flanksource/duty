package models

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Connection", func() {
	ginkgo.Describe("AsGoGetterURL", func() {
		testCases := []struct {
			name          string
			connection    Connection
			expectedURL   string
			expectedError error
		}{
			{
				name: "HTTP Connection",
				connection: Connection{
					Type:     ConnectionTypeHTTP,
					URL:      "http://example.com",
					Username: "testuser",
					Password: "testpassword",
				},
				expectedURL:   "http://testuser:testpassword@example.com",
				expectedError: nil,
			},
			{
				name: "Git Connection",
				connection: Connection{
					Type:        ConnectionTypeGit,
					URL:         "https://github.com/repo.git",
					Certificate: "cert123",
					Properties:  map[string]string{"ref": "main"},
				},
				expectedURL:   "git::https://github.com/repo.git?ref=main&sshkey=Y2VydDEyMw%3D%3D",
				expectedError: nil,
			},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			ginkgo.Context(tc.name, func() {
				ginkgo.It("should return the correct URL and error", func() {
					resultURL, err := tc.connection.AsGoGetterURL()
					Expect(resultURL).To(Equal(tc.expectedURL))
					if tc.expectedError == nil {
						Expect(err).To(BeNil())
					}
				})
			})
		}
	})

	ginkgo.Describe("AsEnv", func() {
		testCases := []struct {
			name                string
			connection          Connection
			expectedEnv         []string
			expectedFileContent string
		}{
			{
				name: "AWS Connection",
				connection: Connection{
					Type:       ConnectionTypeAWS,
					Username:   "awsuser",
					Password:   "awssecret",
					Properties: map[string]string{"profile": "awsprofile", "region": "us-east-1"},
				},
				expectedEnv: []string{
					"AWS_ACCESS_KEY_ID=awsuser",
					"AWS_SECRET_ACCESS_KEY=awssecret",
					"AWS_DEFAULT_PROFILE=awsprofile",
					"AWS_DEFAULT_REGION=us-east-1",
				},
				expectedFileContent: "[default]\naws_access_key_id = awsuser\naws_secret_access_key = awssecret\nregion = us-east-1\n",
			},
			{
				name: "GCP Connection",
				connection: Connection{
					Type:        ConnectionTypeGCP,
					Username:    "gcpuser",
					Certificate: `{"account": "gcpuser"}`,
				},
				expectedEnv:         []string{},
				expectedFileContent: `{"account": "gcpuser"}`,
			},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			ginkgo.Context(tc.name, func() {
				ginkgo.It("should return the correct environment variables and file content", func() {
					envPrep := tc.connection.AsEnv(context.Background())

					for i, expected := range tc.expectedEnv {
						Expect(envPrep.Env[i]).To(Equal(expected))
					}

					for _, content := range envPrep.Files {
						Expect(content.String()).To(Equal(tc.expectedFileContent))
					}
				})
			})
		}
	})
})
