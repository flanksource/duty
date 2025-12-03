package tests

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/dataquery"
	"github.com/flanksource/duty/models"
)

var _ = ginkgo.Describe("SQL Data Query", ginkgo.Ordered, func() {
	var (
		connectionName string
		conn           models.Connection
	)

	ginkgo.BeforeAll(func() {
		conn = models.Connection{
			Name:      "sql-dataquery",
			Namespace: "default",
			Type:      models.ConnectionTypePostgres,
			URL:       DefaultContext.Value("db_url").(string),
			Source:    models.SourceUI,
		}
		Expect(DefaultContext.DB().Save(&conn).Error).ToNot(HaveOccurred())

		connectionName = fmt.Sprintf("connection://%s/%s", conn.Namespace, conn.Name)
	})

	ginkgo.AfterAll(func() {
		Expect(DefaultContext.DB().Delete(&conn).Error).ToNot(HaveOccurred())
	})

	ginkgo.It("hydrates the connection string and queries information_schema", func() {
		query := dataquery.Query{
			SQL: &dataquery.SQLQuery{
				SQLConnection: connection.SQLConnection{
					ConnectionName: connectionName,
				},
				Query: `
					SELECT table_name, table_schema, table_type
					FROM information_schema.tables
					WHERE table_schema = 'public'
					ORDER BY table_name
					LIMIT 5
				`,
			},
		}

		results, err := dataquery.ExecuteQuery(DefaultContext, query)
		Expect(err).ToNot(HaveOccurred())
		Expect(results).ToNot(BeEmpty())
		Expect(len(results)).To(Equal(5))

		for _, row := range results {
			Expect(row).To(HaveKey("table_schema"))
			Expect(row).To(HaveKey("table_name"))
			Expect(row).To(HaveKey("table_type"))

			Expect(fmt.Sprint(row["table_schema"])).To(Equal("public"))
			Expect(fmt.Sprint(row["table_name"])).ToNot(BeEmpty())
			Expect(fmt.Sprint(row["table_type"])).ToNot(BeEmpty())
		}
	})
})
