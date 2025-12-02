package tests

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/tests/setup"
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
	It("should create a client and execute a query", func() {
		conn := models.Connection{
			Name:      "sql-conn",
			Namespace: "default",
			Type:      models.ConnectionTypePostgres,
			URL:       setup.PgUrl,
			Source:    models.SourceUI,
		}
		Expect(DefaultContext.DB().Create(&conn).Error).ToNot(HaveOccurred())
		defer DefaultContext.DB().Delete(&conn)

		sqlConn := connection.SQLConnection{
			ConnectionName: fmt.Sprintf("connection://%s/%s", conn.Namespace, conn.Name),
		}
		Expect(sqlConn.HydrateConnection(DefaultContext)).To(Succeed())

		client, err := sqlConn.Client(DefaultContext)
		Expect(err).ToNot(HaveOccurred())
		defer sqlConn.Close()

		type row struct {
			TableName string `gorm:"column:table_name"`
			TableType string `gorm:"column:table_type"`
		}

		rows, err := client.Query("SELECT table_name, table_type FROM information_schema.tables LIMIT 5")
		Expect(err).ToNot(HaveOccurred())
		defer rows.Close()

		var results []row
		for rows.Next() {
			var r row
			Expect(rows.Scan(&r.TableName, &r.TableType)).To(Succeed())
			results = append(results, r)
		}
		Expect(rows.Err()).ToNot(HaveOccurred())
		Expect(len(results)).To(Equal(5))

	})
})
