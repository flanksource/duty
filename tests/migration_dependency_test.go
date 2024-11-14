package tests

import (
	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/migrate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("migration dependency", Ordered, func() {
	It("should have no executable scripts", func() {
		db, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		funcs, views, err := migrate.GetExecutableScripts(db)
		Expect(err).To(BeNil())
		Expect(len(funcs)).To(BeZero())
		Expect(len(views)).To(BeZero())
	})

	It("should get correct executable scripts", func() {
		err := DefaultContext.DB().Exec(`UPDATE migration_logs SET hash = 'dummy' WHERE path = 'drop.sql'`).Error
		Expect(err).To(BeNil())

		db, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		funcs, views, err := migrate.GetExecutableScripts(db)
		Expect(err).To(BeNil())
		Expect(len(funcs)).To(Equal(1))
		Expect(len(views)).To(Equal(1))

		Expect(collections.MapKeys(funcs)).To(Equal([]string{"drop.sql"}))
		Expect(collections.MapKeys(views)).To(Equal([]string{"021_notification.sql"}))

		{
			// run the migrations again to ensure that the hashes are repopulated
			migrate.RunMigrations(db, api.DefaultConfig)

			// at the end, there should be no scrips to apply
			db, err := DefaultContext.DB().DB()
			Expect(err).To(BeNil())

			funcs, views, err := migrate.GetExecutableScripts(db)
			Expect(err).To(BeNil())
			Expect(len(funcs)).To(BeZero())
			Expect(len(views)).To(BeZero())
		}
	})
})
