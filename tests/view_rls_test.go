package tests

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/view"
)

var _ = Describe("View table RLS", Ordered, Serial, func() {
	const tableName = "view_rls_test"

	columns := view.ViewColumnDefList{
		{Name: "id", Type: view.ColumnTypeString, PrimaryKey: true},
		{Name: "name", Type: view.ColumnTypeString},
	}

	BeforeAll(func() {
		if os.Getenv("DUTY_DB_DISABLE_RLS") == "true" {
			Skip("RLS tests are disabled because DUTY_DB_DISABLE_RLS is set to true")
		}
	})

	AfterAll(func() {
		DefaultContext.DB().Exec("DROP TABLE IF EXISTS " + tableName)
	})

	It("should enable RLS on new view tables", func() {
		api.DefaultConfig.DisableRLS = false
		err := view.CreateViewTable(DefaultContext, tableName, columns)
		Expect(err).To(BeNil())

		Expect(viewTableHasRLS(tableName)).To(BeTrue())
		Expect(viewTableHasGrantsPolicy(tableName)).To(BeTrue())
	})

	It("should disable RLS when config changes without schema changes", func() {
		api.DefaultConfig.DisableRLS = true
		err := view.CreateViewTable(DefaultContext, tableName, columns)
		Expect(err).To(BeNil())

		Expect(viewTableHasRLS(tableName)).To(BeFalse())
		Expect(viewTableHasGrantsPolicy(tableName)).To(BeFalse())
	})

	It("should re-enable RLS when config changes back", func() {
		api.DefaultConfig.DisableRLS = false
		err := view.CreateViewTable(DefaultContext, tableName, columns)
		Expect(err).To(BeNil())

		Expect(viewTableHasRLS(tableName)).To(BeTrue())
		Expect(viewTableHasGrantsPolicy(tableName)).To(BeTrue())
	})

	It("should be idempotent when state already matches", func() {
		// Call again with same config — should be a no-op
		err := view.CreateViewTable(DefaultContext, tableName, columns)
		Expect(err).To(BeNil())

		Expect(viewTableHasRLS(tableName)).To(BeTrue())
		Expect(viewTableHasGrantsPolicy(tableName)).To(BeTrue())
	})
})

func viewTableHasRLS(tableName string) bool {
	var hasRLS bool
	err := DefaultContext.DB().Raw(`
		SELECT c.relrowsecurity
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = ? AND c.relkind = 'r'
	`, tableName).Scan(&hasRLS).Error
	Expect(err).To(BeNil())
	return hasRLS
}

func viewTableHasGrantsPolicy(tableName string) bool {
	var exists bool
	err := DefaultContext.DB().Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_policies
			WHERE schemaname = 'public' AND tablename = ? AND policyname = 'view_grants_policy'
		)
	`, tableName).Scan(&exists).Error
	Expect(err).To(BeNil())
	return exists
}
