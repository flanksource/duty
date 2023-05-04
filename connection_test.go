package duty

import (
	"context"
	"testing"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connection", Ordered, func() {
	BeforeAll(func() {
		tx := testutils.TestDB.Save(&models.Connection{
			Name:     "test",
			Type:     "test",
			Username: "configmap://test-cm/foo",
			Password: "secret://test-secret/foo",
			URL:      "sql://db?user=$(username)&password=$(password)",
		})
		Expect(tx.Error).ToNot(HaveOccurred())
	})

	It("username should be looked up from configmap", func() {
		user, err := GetEnvStringFromCache(testutils.TestClient, "configmap://test-cm/foo", "default")
		Expect(err).ToNot(HaveOccurred())
		Expect(user).To(Equal("bar"))

		val, err := GetConfigMapFromCache(testutils.TestClient, "default", "test-cm", "foo")
		Expect(err).ToNot(HaveOccurred())
		Expect(val).To(Equal("bar"))
	})

	var connection *models.Connection
	var err error
	It("should be retrieved successfully", func() {
		connection, err = GetConnection(context.Background(), testutils.TestClient, testutils.TestDB, "test", "test", "default")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should lookup kubernetes secrets", func() {
		Expect(connection.Username).To(Equal("bar"))
		Expect(connection.Password).To(Equal("secret"))
	})

	It("should template out the url", func() {
		Expect(connection.URL).To(Equal("sql://db?user=bar&password=secret"))
	})
})

func TestGetConnectionNameType(t *testing.T) {
	testCases := []struct {
		name       string
		connection string
		expect     struct {
			name           string
			connectionType string
			found          bool
		}
	}{
		{
			name:       "valid connection string",
			connection: "connection://db/mission_control",
			expect: struct {
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
			expect: struct {
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
			expect: struct {
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
			expect: struct {
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
			expect: struct {
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
			expect: struct {
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
			if name != tc.expect.name {
				t.Errorf("expected name %q, but got %q", tc.expect.name, name)
			}
			if connectionType != tc.expect.connectionType {
				t.Errorf("expected connection type %q, but got %q", tc.expect.connectionType, connectionType)
			}
			if found != tc.expect.found {
				t.Errorf("expected found %t, but got %t", tc.expect.found, found)
			}
		})
	}
}
