package duty

import (
	"database/sql"
	"testing"

	. "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/migrate"
	_ "github.com/flanksource/duty/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSchema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Suite")
}

var postgres *EmbeddedPostgres
var url string

func MustDB() *sql.DB {
	db, err := NewDB()
	if err != nil {
		panic(err)
	}
	return db
}

var _ = BeforeSuite(func() {
	postgres = NewDatabase(DefaultConfig().
		Database("test").
		Port(9876))
	url = "postgres://postgres:postgres@localhost:9876/test?sslmode=disable"
	if err := postgres.Start(); err != nil {
		Fail(err.Error())
	}
	logger.Infof("Started postgres on port 9876")
	if pool != nil {
		return
	}
	if _, err := NewPgxPool(url); err != nil {
		Fail(err.Error())
	}
	if _, err := NewDB(); err != nil {
		Fail(err.Error())
	}
})

var _ = AfterSuite(func() {
	logger.Infof("Stopping postgres")
	if err := postgres.Stop(); err != nil {
		Fail(err.Error())
	}
})

var _ = Describe("Schema", func() {
	It("should be able to run migrations", func() {
		logger.Infof("Running migrations against %s", url)
		err := migrate.Migrate(MustDB(), url)
		Expect(err).ToNot(HaveOccurred())
		// run again to ensure idempotency
		err = migrate.Migrate(MustDB(), url)
		Expect(err).ToNot(HaveOccurred())
	})
	It(" Gorm Can connect", func() {
		gorm, err := NewGorm()
		Expect(err).ToNot(HaveOccurred())
		var people int64
		Expect(gorm.Table("people").Count(&people).Error).ToNot(HaveOccurred())
		Expect(people).To(Equal(int64(1)))
	})
})

var _ = Describe("DB", func() {
	It("Can connect", func() {
		db, err := NewDB()
		Expect(err).ToNot(HaveOccurred())
		result, err := db.Exec("SELECT 1")
		Expect(err).ToNot(HaveOccurred())
		affected, err := result.RowsAffected()
		Expect(affected).To(Equal(int64(1)))
	})
})
