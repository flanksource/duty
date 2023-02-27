package duty

import (
	"github.com/flanksource/commons/logger"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Schema", func() {
	ginkgo.It("should be able to run migrations", func() {
		logger.Infof("Running migrations against %s", pgUrl)
		// run migrations again to ensure idempotency
		err := Migrate(pgUrl)
		Expect(err).ToNot(HaveOccurred())
	})
	ginkgo.It("Gorm can connect", func() {
		gormDB, err := NewGorm(pgUrl, DefaultGormConfig())
		Expect(err).ToNot(HaveOccurred())
		var people int64
		Expect(gormDB.Table("people").Count(&people).Error).ToNot(HaveOccurred())
		Expect(people).NotTo(BeZero())
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
