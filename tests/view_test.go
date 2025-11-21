package tests

import (
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/flanksource/duty/view"
)

var _ = ginkgo.Describe("View Tests", ginkgo.Serial, ginkgo.Ordered, func() {
	ginkgo.Describe("InsertViewRows", func() {
		var pipelineView models.View
		var columnDef view.ViewColumnDefList
		var err error
		var newRows []view.Row

		// Column indices for semantic reference
		const (
			REPOSITORY_COLUMN  = 1
			LAST_RUN_COLUMN    = 2
			LAST_RUN_BY_COLUMN = 3
			DURATION_COLUMN    = 4
		)

		ginkgo.BeforeAll(func() {
			pipelineView = createViewTable(DefaultContext, "pipelines")
			populateViewTable(DefaultContext, pipelineView, "pipelines.json")
			columnDef, err = view.GetViewColumnDefs(DefaultContext, pipelineView.Namespace, pipelineView.Name)
			Expect(err).ToNot(HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			DefaultContext.DB().Exec("DELETE FROM view_mc_pipelines")
		})

		ginkgo.It("should insert rows into view table using pipeline fixtures", func() {
			var newRowCount int
			err := DefaultContext.DB().Raw(`SELECT COUNT(*) FROM ` + pipelineView.GeneratedTableName()).Scan(&newRowCount).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(newRowCount).To(Equal(10))

			{
				// Row count should remain the same (upsert, not duplicate)
				populateViewTable(DefaultContext, pipelineView, "pipelines.json")
				err = DefaultContext.DB().Raw(`SELECT COUNT(*) FROM ` + pipelineView.GeneratedTableName()).Scan(&newRowCount).Error
				Expect(err).ToNot(HaveOccurred())
				Expect(newRowCount).To(Equal(10))
			}
		})

		ginkgo.It("should convert into native go types", func() {
			rows, err := view.ReadViewTable(DefaultContext, columnDef, pipelineView.GeneratedTableName(), "")
			Expect(err).ToNot(HaveOccurred())
			Expect(rows).To(HaveLen(10))
			Expect(rows[0][DURATION_COLUMN]).To(BeAssignableToTypeOf(time.Duration(0)))
			Expect(rows[0][LAST_RUN_COLUMN]).To(BeAssignableToTypeOf(time.Time{}))
		})

		ginkgo.It("should handle updates to existing records", func() {
			newRows = []view.Row{
				{
					"Create Release",
					"flanksource/config-db",
					"2025-07-02T17:47:04+05:45",
					"flankbot-updated", // updates an existing row
					1702000000000,
					"failure",
					nil,
				},
				{
					"New pipeline", // a new row
					"flanksource/config-db",
					"2025-07-02T17:53:18+05:45",
					"flankbot",
					1702000000000,
					"failure",
					nil,
				},
			}

			err := view.InsertViewRows(DefaultContext, pipelineView.GeneratedTableName(), columnDef, newRows, "")
			Expect(err).ToNot(HaveOccurred())

			var newRowCount int
			err = DefaultContext.DB().Raw(`SELECT COUNT(*) FROM ` + pipelineView.GeneratedTableName()).Scan(&newRowCount).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(newRowCount).To(Equal(len(newRows)))

			rows, err := view.ReadViewTable(DefaultContext, columnDef, pipelineView.GeneratedTableName(), "")
			Expect(err).ToNot(HaveOccurred())
			Expect(rows).To(HaveLen(len(newRows)))
			Expect(rows[0][REPOSITORY_COLUMN]).To(Equal(newRows[0][REPOSITORY_COLUMN]), "repository")
			Expect(rows[0][LAST_RUN_BY_COLUMN]).To(Equal(newRows[0][LAST_RUN_BY_COLUMN]), "lastRunBy")
			Expect(rows[1][REPOSITORY_COLUMN]).To(Equal(newRows[1][REPOSITORY_COLUMN]), "repository")
			Expect(rows[1][LAST_RUN_BY_COLUMN]).To(Equal(newRows[1][LAST_RUN_BY_COLUMN]), "lastRunBy")
		})

		ginkgo.It("should handle updates to the column order in view definition", func() {
			// When the column order changes or a new column is added, this test ensures that the records
			// are read in the order the columns are defined in the view spec and not in the order they are
			// stored in the database.

			// Switch the order of `repository` and `lastRunBy` columns
			columnDef[1], columnDef[3] = columnDef[3], columnDef[1]

			// After swapping, the column indices are now reversed
			const (
				SWAPPED_REPOSITORY_COLUMN  = 3 // repository is now at 4th column
				SWAPPED_LAST_RUN_BY_COLUMN = 1 // lastRunBy is now at 2nd column
			)

			rows, err := view.ReadViewTable(DefaultContext, columnDef, pipelineView.GeneratedTableName(), "")
			Expect(err).ToNot(HaveOccurred())
			Expect(rows).To(HaveLen(2))
			Expect(rows[0][SWAPPED_REPOSITORY_COLUMN]).To(Equal(newRows[0][REPOSITORY_COLUMN]), "repository is the 4th column")
			Expect(rows[0][SWAPPED_LAST_RUN_BY_COLUMN]).To(Equal(newRows[0][LAST_RUN_BY_COLUMN]), "lastRunBy is the 2nd column")
			Expect(rows[1][SWAPPED_REPOSITORY_COLUMN]).To(Equal(newRows[1][REPOSITORY_COLUMN]), "repository is the 4th column")
			Expect(rows[1][SWAPPED_LAST_RUN_BY_COLUMN]).To(Equal(newRows[1][LAST_RUN_BY_COLUMN]), "lastRunBy is the 2nd column")
		})

		ginkgo.It("should handle empty rows by clearing the table", func() {
			err := view.InsertViewRows(DefaultContext, pipelineView.GeneratedTableName(), columnDef, []view.Row{}, "")
			Expect(err).ToNot(HaveOccurred())

			// Verify table is now empty
			var newRowCount int
			err = DefaultContext.DB().Raw(`SELECT COUNT(*) FROM ` + pipelineView.GeneratedTableName()).Scan(&newRowCount).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(newRowCount).To(BeZero())
		})
	})

	ginkgo.Describe("CreateViewTable", func() {
		var testView models.View
		var testTableName string

		ginkgo.BeforeAll(func() {
			testView = createViewTable(DefaultContext, "test_grants")
			testTableName = testView.GeneratedTableName()
		})

		ginkgo.AfterAll(func() {
			DefaultContext.DB().Exec("DROP TABLE IF EXISTS " + testTableName)
		})

		ginkgo.It("should create __grants column in view table", func() {
			var columnExists bool
			err := DefaultContext.DB().Raw(`
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = ? AND column_name = '__grants'
				)
			`, testTableName).Scan(&columnExists).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(columnExists).To(BeTrue(), "__grants column should exist in view table")
		})

		ginkgo.It("should enable RLS on view table", func() {
			var rlsEnabled bool
			err := DefaultContext.DB().Raw(`
        SELECT relrowsecurity
        FROM pg_class
        WHERE relname = ? AND relkind = 'r'
    `, testTableName).Scan(&rlsEnabled).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(rlsEnabled).To(BeTrue(), "RLS should be enabled on view table")
		})

		ginkgo.It("should create view_grants_policy on view table", func() {
			var policyExists bool
			err := DefaultContext.DB().Raw(`
				SELECT EXISTS (
					SELECT 1 FROM pg_policies
					WHERE tablename = ? AND policyname = 'view_grants_policy'
				)
			`, testTableName).Scan(&policyExists).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(policyExists).To(BeTrue(), "view_grants_policy should exist on view table")
		})
	})

	ginkgo.Describe("ColumnURL.Eval", func() {
		var configItem = dummy.NginxHelmRelease
		env := map[string]any{
			"row": configItem.AsMap(),
		}

		ginkgo.It("should handle UUID config values", func() {
			colURL := view.ColumnURL{
				Config: types.CelExpression(`row.id`),
			}

			result, err := colURL.Eval(DefaultContext, env)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("/catalog/" + configItem.ID.String()))
		})

		ginkgo.It("should handle resource selector config values", func() {
			colURL := view.ColumnURL{
				// This can be row.id but we're testing the search query functionality
				Config: types.CelExpression(`f("name=$(name) type=$(type)", row)`),
			}

			result, err := colURL.Eval(DefaultContext, env)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("/catalog/" + configItem.ID.String()))
		})

		ginkgo.It("should NOT return error for resource selector with no matches", func() {
			colURL := view.ColumnURL{
				Config: types.CelExpression(`"name=nonexistent-config"`),
			}

			result, err := colURL.Eval(DefaultContext, env)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
