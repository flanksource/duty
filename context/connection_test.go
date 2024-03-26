package context

import (
	"context"
	"testing"

	commons "github.com/flanksource/commons/context"
	"github.com/flanksource/duty/models"
	"github.com/google/go-cmp/cmp"
)

func TestGetConnectionNameType(t *testing.T) {
	testCases := []struct {
		name       string
		connection string
		Expect     struct {
			name           string
			namespace      string
			connectionType string
			found          bool
		}
	}{
		{
			name:       "valid connection string",
			connection: "connection://db/default/mission_control",
			Expect: struct {
				name           string
				namespace      string
				connectionType string
				found          bool
			}{
				name:           "mission_control",
				namespace:      "default",
				connectionType: "db",
				found:          true,
			},
		},
		{
			name:       "valid connection string | name has /",
			connection: "connection://db/default/mission_control//",
			Expect: struct {
				name           string
				namespace      string
				connectionType string
				found          bool
			}{
				name:           "",
				namespace:      "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "invalid | host only",
			connection: "connection:///type-only",
			Expect: struct {
				name           string
				namespace      string
				connectionType string
				found          bool
			}{
				name:           "",
				namespace:      "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "invalid connection string",
			connection: "invalid-connection-string",
			Expect: struct {
				name           string
				namespace      string
				connectionType string
				found          bool
			}{
				name:           "",
				namespace:      "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "empty connection string",
			connection: "",
			Expect: struct {
				name           string
				namespace      string
				connectionType string
				found          bool
			}{
				name:           "",
				namespace:      "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "connection string with type only",
			connection: "connection://type-only",
			Expect: struct {
				name           string
				namespace      string
				connectionType string
				found          bool
			}{
				name:           "",
				namespace:      "",
				connectionType: "",
				found:          false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, namespace, connectionType, found := extractConnectionNameType(tc.connection)
			if name != tc.Expect.name {
				t.Errorf("g.Expected name %q, but got %q", tc.Expect.name, name)
			}
			if namespace != tc.Expect.namespace {
				t.Errorf("g.Expected namespace %q, but got %q", tc.Expect.namespace, namespace)
			}
			if connectionType != tc.Expect.connectionType {
				t.Errorf("g.Expected connection type %q, but got %q", tc.Expect.connectionType, connectionType)
			}
			if found != tc.Expect.found {
				t.Errorf("g.Expected found %t, but got %t", tc.Expect.found, found)
			}
		})
	}
}

func TestHydrateConnection(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := HydrateConnection(dummyContext, &tc.connection)
			if err != nil {
				t.Fatalf("%v", err)
			}

			if diff := cmp.Diff(tc.expect, resp.URL); diff != "" {
				t.Logf("mismatch: wanted=%s got=%s", tc.expect, resp.URL)
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
