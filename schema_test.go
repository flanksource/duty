package duty

import (
	"database/sql"
	"io"
	"testing"

	. "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	_ "github.com/flanksource/duty/types"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSchema(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Schema Suite")
}

var postgres *EmbeddedPostgres

const pgUrl = "postgres://postgres:postgres@localhost:9876/test?sslmode=disable"

func MustDB() *sql.DB {
	db, err := NewDB(pgUrl)
	if err != nil {
		panic(err)
	}
	return db
}

var _ = ginkgo.BeforeSuite(func() {
	postgres = NewDatabase(DefaultConfig().
		Database("test").
		Port(9876).
		Logger(io.Discard))
	if err := postgres.Start(); err != nil {
		ginkgo.Fail(err.Error())
	}
	logger.Infof("Started postgres on port 9876")
	if pool != nil {
		return
	}
	if _, err := NewPgxPool(pgUrl); err != nil {
		ginkgo.Fail(err.Error())
	}
	if _, err := NewDB(pgUrl); err != nil {
		ginkgo.Fail(err.Error())
	}
	err := Migrate(pgUrl)
	Expect(err).ToNot(HaveOccurred())

})

var _ = ginkgo.AfterSuite(func() {
	logger.Infof("Stopping postgres")
	if err := postgres.Stop(); err != nil {
		ginkgo.Fail(err.Error())
	}
})

var _ = ginkgo.Describe("Schema", func() {
	ginkgo.It("should be able to run migrations", func() {
		logger.Infof("Running migrations against %s", pgUrl)
		// run again to ensure idempotency
		err := Migrate(pgUrl)
		Expect(err).ToNot(HaveOccurred())
	})
	ginkgo.It(" Gorm Can connect", func() {
		gorm, err := NewGorm(pgUrl, DefaultGormConfig())
		Expect(err).ToNot(HaveOccurred())
		var people int64
		Expect(gorm.Table("people").Count(&people).Error).ToNot(HaveOccurred())
		Expect(people).To(Equal(int64(1)))
	})
})

var _ = ginkgo.Describe("DB", func() {
	ginkgo.It("Can connect", func() {
		db, err := NewDB(pgUrl)
		Expect(err).ToNot(HaveOccurred())
		result, err := db.Exec("SELECT 1")
		Expect(err).ToNot(HaveOccurred())
		affected, _ := result.RowsAffected()
		Expect(affected).To(Equal(int64(1)))
	})
})
