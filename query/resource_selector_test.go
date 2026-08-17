package query

import (
	"testing"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/lib/pq"
	"github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestConfigExternalIDFieldSelectorUsesTextArrayColumn(t *testing.T) {
	g := gomega.NewWithT(t)
	db, err := gorm.Open(postgres.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ctx := context.New().WithDB(db, nil)
	tx, err := SetResourceSelectorClause(ctx, types.ResourceSelector{
		Agent:          "all",
		IncludeDeleted: true,
		Types:          []string{"Kubernetes::Pod"},
		FieldSelector:  "external_id=  MiXeD-Alias  ",
	}, db.Table(models.ConfigItem{}.TableName()), models.ConfigItem{}.TableName())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	statement := tx.Find(&[]models.ConfigItem{}).Statement
	g.Expect(statement.SQL.String()).To(gomega.Equal(
		`SELECT * FROM "config_items" WHERE LOWER(CAST("type" AS TEXT)) = $1 AND "external_id" @> $2::text[]`,
	))
	g.Expect(statement.Vars).To(gomega.Equal([]any{
		"kubernetes::pod",
		pq.StringArray{"mixed-alias"},
	}))
}
