package tests

import (
	"os"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/tests/setup"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Schema", ginkgo.Label("slow"), func() {
	ginkgo.It("should be able to run migrations", func() {
		logger.Infof("Running migrations against %s", setup.PgUrl)
		// run migrations again to ensure idempotency
		conf := api.NewConfig(setup.PgUrl)

		// Respect the DUTY_DB_DISABLE_RLS environment variable
		if os.Getenv("DUTY_DB_DISABLE_RLS") == "true" {
			conf.DisableRLS = true
		} else {
			conf.EnableRLS = true
		}

		err := duty.Migrate(conf)
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.It("Gorm can connect", func() {
		gormDB, err := duty.NewGorm(setup.PgUrl, duty.DefaultGormConfig())
		Expect(err).ToNot(HaveOccurred())
		var people int64
		Expect(gormDB.Table("people").Count(&people).Error).ToNot(HaveOccurred())
		Expect(people).NotTo(BeZero())
	})
})

var _ = ginkgo.Describe("DB", func() {
	ginkgo.It("Can connect", func() {
		db, err := duty.NewDB(setup.PgUrl)
		Expect(err).ToNot(HaveOccurred())
		result, err := db.Exec("SELECT 1")
		Expect(err).ToNot(HaveOccurred())
		affected, _ := result.RowsAffected()
		Expect(affected).To(Equal(int64(1)))
	})
})
