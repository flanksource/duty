package tests

import (
	"github.com/flanksource/duty/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connection", Ordered, func() {
	var azureconnection = models.Connection{
		Name:      "azure-dev",
		Type:      "postgresql",
		Namespace: "mission-control",
		Username:  "username",
		Password:  "password",
		Source:    models.SourceCRD,
		URL:       "sql://db?user=$(username)&password=$(password)",
	}

	var testConnection = models.Connection{
		Name:      "test",
		Type:      "test",
		Namespace: "default",
		Username:  "configmap://test-cm/foo",
		Password:  "secret://test-secret/foo",
		Source:    models.SourceCRD,
		URL:       "sql://db?user=$(username)&password=$(password)",
	}

	BeforeAll(func() {
		tx := DefaultContext.DB().Save(&testConnection)
		Expect(tx.Error).ToNot(HaveOccurred())

		tx = DefaultContext.DB().Save(&azureconnection)
		Expect(tx.Error).ToNot(HaveOccurred())
	})

	It("username should be looked up from configmap", func() {
		user, err := DefaultContext.GetEnvStringFromCache("configmap://test-cm/foo", "default")
		Expect(err).ToNot(HaveOccurred())
		Expect(user).To(Equal("bar"))

		val, err := DefaultContext.GetConfigMapFromCache("default", "test-cm", "foo")
		Expect(err).ToNot(HaveOccurred())
		Expect(val).To(Equal("bar"))
	})

	Describe("fetching", func() {
		It("old format for backward compatibility", func() {
			connectionString := "connection://postgresql/azure-dev"
			con, err := DefaultContext.HydrateConnectionByURL(connectionString)
			Expect(err).To(BeNil())
			Expect(con).To(Not(BeNil()))

			Expect(con.ID).To(Equal(azureconnection.ID))
		})

		It("new format with namespace", func() {
			connectionString := "connection://mission-control/azure-dev"
			con, err := DefaultContext.HydrateConnectionByURL(connectionString)
			Expect(err).To(BeNil())
			Expect(con).To(Not(BeNil()))
			Expect(con.ID).To(Equal(azureconnection.ID))
		})

		It("new format with just the name", func() {
			connectionString := "connection://azure-dev"
			con, err := DefaultContext.WithNamespace("mission-control").HydrateConnectionByURL(connectionString)
			Expect(err).To(BeNil())
			Expect(con).To(Not(BeNil()))
			Expect(con.ID).To(Equal(azureconnection.ID))
		})
	})

	var connection *models.Connection
	var err error
	It("should be retrieved successfully", func() {
		connection, err = DefaultContext.GetConnection("test", "default")
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
