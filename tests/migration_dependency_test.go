package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/migrate"
)

var _ = Describe("migration dependency", Ordered, Serial, func() {
	var connString string

	BeforeAll(func() {
		connString = DefaultContext.Value("db_url").(string)
	})

	AfterAll(func() {
		sqlDB, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		// we re-enable RLS
		err = migrate.RunMigrations(sqlDB, api.Config{ConnectionString: connString, EnableRLS: true})
		Expect(err).To(BeNil())
	})

	It("should have no executable scripts", func() {
		db, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		funcs, views, err := migrate.GetExecutableScripts(db, nil, nil)
		Expect(err).To(BeNil())
		Expect(len(funcs)).To(BeZero())
		Expect(len(views)).To(Equal(1), "skipped RLS disable is picked up here")
	})

	It("should explicitly run script", func() {
		db, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		funcs, views, err := migrate.GetExecutableScripts(db, []string{"incident_ids.sql"}, nil)
		Expect(err).To(BeNil())
		Expect(len(funcs)).To(Equal(1))
		Expect(len(views)).To(Equal(1), "skipped RLS disable is picked up here")
	})

	It("should ignore changed hash run script", func() {
		var currentHash string
		err := DefaultContext.DB().Raw(`SELECT hash FROM migration_logs WHERE path = 'incident_ids.sql'`).Scan(&currentHash).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Exec(`UPDATE migration_logs SET hash = 'dummy' WHERE path = 'incident_ids.sql'`).Error
		Expect(err).To(BeNil())

		db, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		funcs, views, err := migrate.GetExecutableScripts(db, nil, []string{"incident_ids.sql"})
		Expect(err).To(BeNil())
		Expect(len(funcs)).To(BeZero())
		Expect(len(views)).To(Equal(1), "skipped RLS disable is picked up here")

		err = DefaultContext.DB().Exec(`UPDATE migration_logs SET hash = ? WHERE path = 'incident_ids.sql'`, []byte(currentHash)[:]).Error
		Expect(err).To(BeNil(), "failed to restore hash for incidents_ids.sql")
	})

	It("should get correct executable scripts", func() {
		err := DefaultContext.DB().Exec(`UPDATE migration_logs SET hash = 'dummy' WHERE path = 'drop.sql'`).Error
		Expect(err).To(BeNil())

		sqlDB, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		funcs, views, err := migrate.GetExecutableScripts(sqlDB, nil, []string{"034_rls_enable.sql", "035_rls_disable.sql"})
		Expect(err).To(BeNil())
		Expect(len(funcs)).To(Equal(1))
		Expect(len(views)).To(Equal(2))

		Expect(collections.MapKeys(funcs)).To(Equal([]string{"drop.sql"}))
		Expect(collections.MapKeys(views)).To(ConsistOf([]string{"006_config_views.sql", "021_notification.sql"}))

		{
			// run the migrations again to ensure that the hashes are repopulated
			err := migrate.RunMigrations(sqlDB, api.Config{ConnectionString: connString, DisableRLS: true})
			Expect(err).To(BeNil())

			// at the end, there should be no scripts to apply
			db, err := DefaultContext.DB().DB()
			Expect(err).To(BeNil())

			funcs, views, err := migrate.GetExecutableScripts(db, nil, nil)
			Expect(err).To(BeNil())
			Expect(len(funcs)).To(BeZero())
			Expect(len(views)).To(BeZero())
		}
	})
})
