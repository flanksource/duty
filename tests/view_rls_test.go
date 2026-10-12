package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/view"
)

var _ = Describe("View table RLS", Ordered, Serial, func() {
	const tableName = "view_rls_test"

	columns := view.ViewColumnDefList{
		{Name: "id", Type: view.ColumnTypeString, PrimaryKey: true},
		{Name: "name", Type: view.ColumnTypeString},
	}

	AfterAll(func() {
		DefaultContext.DB().Exec("DROP TABLE IF EXISTS " + tableName)
	})

	It("should enable RLS on new view tables", func() {
		err := view.CreateViewTable(DefaultContext, tableName, columns)
		Expect(err).To(BeNil())

		Expect(viewTableHasRLS(tableName)).To(BeTrue())
		Expect(viewTableHasGrantsPolicy(tableName)).To(BeTrue())
	})

	It("should re-enable RLS on a view table that has it disabled", func() {
		Expect(DefaultContext.DB().Exec("DROP POLICY view_grants_policy ON " + tableName).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("ALTER TABLE " + tableName + " DISABLE ROW LEVEL SECURITY").Error).To(Succeed())

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
