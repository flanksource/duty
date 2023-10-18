package models

import (
	"testing"
)

func Test_Connection_AsGoGetterURL(t *testing.T) {
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
			expectedURL:   "https://github.com/repo.git?ref=main&sshkey=cert123",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resultURL, err := tc.connection.AsGoGetterURL()

			if resultURL != tc.expectedURL {
				t.Errorf("Expected URL: %s, but got: %s", tc.expectedURL, resultURL)
			}

			if err != tc.expectedError {
				t.Errorf("Expected error: %v, but got: %v", tc.expectedError, err)
			}
		})
	}
}

func Test_Connection_AsEnv(t *testing.T) {
	testCases := []struct {
		name         string
		connection   Connection
		expectedEnv  []string
		expectedFile string
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
			expectedFile: "[default]\naws_access_key_id = awsuser\naws_secret_access_key = awssecret\nregion = us-east-1\n",
		},
		{
			name: "GCP Connection",
			connection: Connection{
				Type:        ConnectionTypeGCP,
				Username:    "gcpuser",
				Certificate: `{"account": "gcpuser"}`,
			},
			expectedEnv:  []string{},
			expectedFile: `{"account": "gcpuser"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env, file := tc.connection.AsEnv()

			for i, expected := range tc.expectedEnv {
				if env[i] != expected {
					t.Errorf("Expected environment variable: %s, but got: %s", expected, env[i])
				}
			}

			if file != tc.expectedFile {
				t.Errorf("Expected file content:\n%s\nBut got:\n%s", tc.expectedFile, file)
			}
		})
	}
}
