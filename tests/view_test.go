package tests

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/view"
)

var _ = ginkgo.Describe("View Tests", ginkgo.Serial, ginkgo.Ordered, func() {
	ginkgo.Describe("InsertViewRows", func() {
		var pipelineView models.View
		var columns view.ViewColumnDefList
		var err error

		ginkgo.BeforeAll(func() {
			pipelineView = createViewTable(DefaultContext, "pipelines")
			populateViewTable(DefaultContext, pipelineView, "pipelines.json")
			columns, err = view.GetViewColumnDefs(DefaultContext, pipelineView.Namespace, pipelineView.Name)
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

		ginkgo.It("should handle updates", func() {
			newRows := []view.Row{
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

			err := view.InsertViewRows(DefaultContext, pipelineView.GeneratedTableName(), columns, newRows)
			Expect(err).ToNot(HaveOccurred())

			// Verify table is now empty
			var newRowCount int
			err = DefaultContext.DB().Raw(`SELECT COUNT(*) FROM ` + pipelineView.GeneratedTableName()).Scan(&newRowCount).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(newRowCount).To(Equal(len(newRows)))

			type Row struct {
				Repository string `gorm:"column:repository"`
				Lastrunby  string `gorm:"column:lastRunBy"`
			}

			var repo []Row
			err = DefaultContext.DB().Table(pipelineView.GeneratedTableName()).Select(`"repository", "lastRunBy"`).Scan(&repo).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(repo).To(ConsistOf(
				Row{
					Repository: "flanksource/config-db",
					Lastrunby:  "flankbot-updated",
				},
				Row{
					Repository: "flanksource/config-db",
					Lastrunby:  "flankbot",
				},
			))
		})

		ginkgo.It("should handle empty rows by clearing the table", func() {
			err := view.InsertViewRows(DefaultContext, pipelineView.GeneratedTableName(), columns, []view.Row{})
			Expect(err).ToNot(HaveOccurred())

			// Verify table is now empty
			var newRowCount int
			err = DefaultContext.DB().Raw(`SELECT COUNT(*) FROM ` + pipelineView.GeneratedTableName()).Scan(&newRowCount).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(newRowCount).To(BeZero())
		})
	})
})
