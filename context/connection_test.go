package context

import "testing"

func TestGetConnectionNameType(t *testing.T) {
	testCases := []struct {
		name       string
		connection string
		Expect     struct {
			name           string
			connectionType string
			found          bool
		}
	}{
		{
			name:       "valid connection string",
			connection: "connection://db/mission_control",
			Expect: struct {
				name           string
				connectionType string
				found          bool
			}{
				name:           "mission_control",
				connectionType: "db",
				found:          true,
			},
		},
		{
			name:       "valid connection string | name has /",
			connection: "connection://db/mission_control//",
			Expect: struct {
				name           string
				connectionType string
				found          bool
			}{
				name:           "mission_control//",
				connectionType: "db",
				found:          true,
			},
		},
		{
			name:       "invalid | host only",
			connection: "connection:///type-only",
			Expect: struct {
				name           string
				connectionType string
				found          bool
			}{
				name:           "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "invalid connection string",
			connection: "invalid-connection-string",
			Expect: struct {
				name           string
				connectionType string
				found          bool
			}{
				name:           "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "empty connection string",
			connection: "",
			Expect: struct {
				name           string
				connectionType string
				found          bool
			}{
				name:           "",
				connectionType: "",
				found:          false,
			},
		},
		{
			name:       "connection string with type only",
			connection: "connection://type-only",
			Expect: struct {
				name           string
				connectionType string
				found          bool
			}{
				name:           "",
				connectionType: "",
				found:          false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, connectionType, found := extractConnectionNameType(tc.connection)
			if name != tc.Expect.name {
				t.Errorf("g.Expected name %q, but got %q", tc.Expect.name, name)
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
