package tests

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = Describe("Exec Connection", Ordered, func() {
	Context("fromConfigItem", func() {
		It("should error out early with no db error", func() {
			txError := DefaultContext.DB().Transaction(func(tx *gorm.DB) error {
				execConnection := connection.ExecConnections{
					FromConfigItem: lo.ToPtr("$(.config.id)"),
				}

				cmd := exec.Cmd{}
				_, err := connection.SetupConnection(DefaultContext, execConnection, &cmd)
				Expect(err).To(Not(BeNil()))
				Expect(err.Error()).To(ContainSubstring("is not a valid uuid"))

				return nil
			})
			Expect(txError).To(BeNil())
		})

		It("should error out early with no db error", func() {
			txError := DefaultContext.DB().Transaction(func(tx *gorm.DB) error {
				execConnection := connection.ExecConnections{
					FromConfigItem: lo.ToPtr(dummy.EKSCluster.ID.String()), // has no scraper
				}

				cmd := exec.Cmd{}
				_, err := connection.SetupConnection(DefaultContext, execConnection, &cmd)
				Expect(err).To(Not(BeNil()))
				Expect(err.Error()).To(ContainSubstring("config item does not have a scraper"))

				return nil
			})
			Expect(txError).To(BeNil())
		})

		It("should setup kubeconfig on the cmd environment", func() {
			txError := DefaultContext.DB().Transaction(func(tx *gorm.DB) error {
				execConnection := connection.ExecConnections{
					FromConfigItem: lo.ToPtr(dummy.KubernetesCluster.ID.String()), // has a scraper
				}

				cmd := exec.Cmd{}
				_, err := connection.SetupConnection(DefaultContext, execConnection, &cmd)
				Expect(err).To(BeNil())
				Expect(cmd.Env[0]).To(Equal("KUBECONFIG=testdata/my-kube-config.yaml"))

				return nil
			})
			Expect(txError).To(BeNil())
		})
	})
})

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

var _ = Describe("SQLConnection", func() {
	It("should convert to/from model", func() {
		model := models.Connection{
			Name:     "sql-conn",
			Type:     models.ConnectionTypePostgres,
			URL:      "postgres://localhost:5432/db",
			Username: "user",
			Password: "pass",
		}

		var sqlConn connection.SQLConnection
		Expect(sqlConn.FromModel(model)).To(Succeed())

		roundTripped := sqlConn.ToModel()
		Expect(roundTripped.Name).To(Equal(model.Name))
		Expect(roundTripped.Type).To(Equal(model.Type))
		Expect(roundTripped.URL).To(Equal(model.URL))
		Expect(roundTripped.Username).To(Equal(model.Username))
		Expect(roundTripped.Password).To(Equal(model.Password))
		Expect(roundTripped.Properties).To(HaveKeyWithValue("sslmode", "false"))
	})

	It("should map sslmode flag to/from properties", func() {
		model := models.Connection{
			Name:       "sql-conn-ssl",
			Type:       models.ConnectionTypePostgres,
			URL:        "postgres://localhost:5432/db",
			Properties: map[string]string{"sslmode": "true"},
		}

		var sqlConn connection.SQLConnection
		Expect(sqlConn.FromModel(model)).To(Succeed())
		Expect(sqlConn.SSLMode).To(BeTrue())

		roundTripped := sqlConn.ToModel()
		Expect(roundTripped.Properties).To(HaveKeyWithValue("sslmode", "true"))
	})
})
