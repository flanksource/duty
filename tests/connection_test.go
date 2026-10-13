package tests

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/tests/setup"
	"github.com/flanksource/duty/types"
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

var _ = Describe("Connection credential views", func() {
	It("masks arbitrary properties before projection, filtering or embedding by API roles", func() {
		conn := models.Connection{
			Name: "view-credential-regression", Namespace: "default", Type: models.ConnectionTypeHTTP, Source: models.SourceUI,
			Username: "view-user", Password: "view-password", Certificate: "view-certificate",
			URL: "https://host/webhook-secret?token=query-secret", InsecureTLS: true,
			Properties: types.JSONStringMap{
				"bearer": "view-bearer", "custom": "view-custom-secret", "region": "eu-west-1",
				"oauth": `{"clientSecret":{"value":"view-oauth-secret"}}`,
				"tls":   `{"key":{"value":"view-key"}}`, "reference": "secret://creds/token",
			},
		}
		Expect(DefaultContext.DB().Create(&conn).Error).To(Succeed())
		defer DefaultContext.DB().Delete(&conn)
		pool, err := DefaultContext.DB().DB()
		Expect(err).ToNot(HaveOccurred())
		cfg := api.Config{ConnectionString: DefaultContext.Value("db_url").(string), Postgrest: api.PostgrestConfig{DBRole: "postgrest_api", AnonDBRole: "postgrest_anon"}}
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		for _, role := range []string{"postgrest_api", "postgrest_anon"} {
			tx := DefaultContext.DB().Begin()
			Expect(tx.Exec("SET LOCAL ROLE " + role).Error).To(Succeed())
			var visible models.Connection
			Expect(tx.Table("connection_details").Where("id = ?", conn.ID).Take(&visible).Error).To(Succeed())
			Expect(visible.Name).To(Equal(conn.Name))
			Expect(visible.Type).To(Equal(conn.Type))
			Expect(visible.InsecureTLS).To(BeTrue())
			Expect(visible.Username).To(Equal("***"))
			Expect(visible.Password).To(Equal("***"))
			Expect(visible.Certificate).To(Equal("***"))
			Expect(visible.URL).To(BeEmpty())
			for _, key := range []string{"bearer", "custom", "region", "oauth", "tls"} {
				Expect(visible.Properties).To(HaveKeyWithValue(key, "***"))
			}
			Expect(visible.Properties).To(HaveKeyWithValue("reference", "secret://creds/token"))
			var projected string
			Expect(tx.Raw("SELECT properties->>'bearer' FROM connection_details WHERE id = ?", conn.ID).Scan(&projected).Error).To(Succeed())
			Expect(projected).To(Equal("***"))
			var count int64
			Expect(tx.Table("connection_details").Where("id = ? AND properties->>'bearer' = ?", conn.ID, "view-bearer").Count(&count).Error).To(Succeed())
			Expect(count).To(BeZero())
			var embedded string
			Expect(tx.Raw("SELECT row_to_json(c)::text FROM connection_details c WHERE id = ?", conn.ID).Scan(&embedded).Error).To(Succeed())
			Expect(embedded).ToNot(ContainSubstring("view-bearer"))
			Expect(tx.Table("connections_list").Where("id = ?", conn.ID).Take(&visible).Error).To(Succeed())
			Expect(tx.Raw("SELECT * FROM connections WHERE id = ?", conn.ID).Scan(&visible).Error).To(HaveOccurred())
			Expect(tx.Rollback().Error).To(Succeed())
		}
		var stored models.Connection
		Expect(DefaultContext.DB().First(&stored, "id = ?", conn.ID).Error).To(Succeed())
		Expect(stored.Properties).To(Equal(conn.Properties))
		Expect(stored.Password).To(Equal(conn.Password))
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
		defer client.Close()

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
