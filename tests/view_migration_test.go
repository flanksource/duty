package tests

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	pkgView "github.com/flanksource/duty/view"
)

var _ = ginkgo.Describe("View Migration Tests", ginkgo.Serial, ginkgo.Ordered, func() {
	var testView models.View

	// Create initial column definitions
	initialColumns := pkgView.ViewColumnDefList{
		{Name: "id", Type: pkgView.ColumnTypeString, PrimaryKey: true},
		{Name: "name", Type: pkgView.ColumnTypeString},
		{Name: "status", Type: pkgView.ColumnTypeString},
		{Name: "created_at", Type: pkgView.ColumnTypeDateTime},
	}

	// Create updated column definitions with new columns
	updatedColumns := pkgView.ViewColumnDefList{
		{Name: "id", Type: pkgView.ColumnTypeString, PrimaryKey: true},
		{Name: "name", Type: pkgView.ColumnTypeString},
		{Name: "status", Type: pkgView.ColumnTypeString},
		{Name: "created_at", Type: pkgView.ColumnTypeDateTime},
		{Name: "priority", Type: pkgView.ColumnTypeNumber},    // New column
		{Name: "description", Type: pkgView.ColumnTypeString}, // New column
	}

	// Remove the status column
	removedColumns := pkgView.ViewColumnDefList{
		{Name: "id", Type: pkgView.ColumnTypeString, PrimaryKey: true},
		{Name: "name", Type: pkgView.ColumnTypeString},
		{Name: "created_at", Type: pkgView.ColumnTypeDateTime},
		{Name: "priority", Type: pkgView.ColumnTypeNumber},
		{Name: "description", Type: pkgView.ColumnTypeString},
	}

	// Add email as primary key (backward incompatible - email column has NULL values in existing data)
	backwardIncompatibleColumns := pkgView.ViewColumnDefList{
		{Name: "id", Type: pkgView.ColumnTypeString},
		{Name: "name", Type: pkgView.ColumnTypeString},
		{Name: "email", Type: pkgView.ColumnTypeString, PrimaryKey: true},
		{Name: "created_at", Type: pkgView.ColumnTypeDateTime},
		{Name: "priority", Type: pkgView.ColumnTypeNumber},
		{Name: "description", Type: pkgView.ColumnTypeString},
	}

	ginkgo.BeforeAll(func() {
		testView = models.View{
			ID:        uuid.New(),
			Name:      "my_table",
			Namespace: "default",
			Spec:      types.JSON("{}"),
			Source:    models.SourceApplicationCRD,
		}

		err := DefaultContext.DB().Create(&testView).Error
		Expect(err).ToNot(HaveOccurred())

		migrateToNewColumns(DefaultContext, testView, initialColumns)

		{
			testData := []pkgView.Row{
				{"test-1", "First Test", "active", "2023-01-01T00:00:00Z"},
				{"test-2", "Second Test", "inactive", "2023-01-02T00:00:00Z"},
			}

			err = pkgView.InsertViewRows(DefaultContext, testView.GeneratedTableName(), initialColumns, testData, "")
			Expect(err).ToNot(HaveOccurred())

			var count int64
			err = DefaultContext.DB().Table(testView.GeneratedTableName()).Count(&count).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(2)))
		}
	})

	ginkgo.AfterAll(func() {
		err := DefaultContext.DB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", testView.GeneratedTableName())).Error
		Expect(err).ToNot(HaveOccurred())

		err = DefaultContext.DB().Delete(&testView).Error
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.Describe("Update view schema and recreate table", func() {
		ginkgo.It("should update the view spec with new columns", func() {
			migrateToNewColumns(DefaultContext, testView, updatedColumns)
		})

		ginkgo.It("should update the view with removed columns", func() {
			migrateToNewColumns(DefaultContext, testView, removedColumns)
		})

		ginkgo.It("should drop and recreate table when primary key change fails", func() {
			migrateToNewColumns(DefaultContext, testView, backwardIncompatibleColumns)
		})
	})
})

func migrateToNewColumns(ctx context.Context, view models.View, columns pkgView.ViewColumnDefList) {
	spec := map[string]any{
		"columns": columns,
	}
	specBytes, err := json.Marshal(spec)
	Expect(err).ToNot(HaveOccurred())

	err = ctx.DB().Model(&view).Update("spec", types.JSON(specBytes)).Error
	Expect(err).ToNot(HaveOccurred())

	err = pkgView.CreateViewTable(ctx, view.GeneratedTableName(), columns)
	Expect(err).ToNot(HaveOccurred())

	// +2 for agent_id and is_pushed + 1 for __row__attributes + 1 for request_fingerprint
	const reservedColumns = 4

	// Fetch all the column names from the table
	var columnNames []string
	err = ctx.DB().Raw("SELECT column_name FROM information_schema.columns WHERE table_name = ?", view.GeneratedTableName()).Scan(&columnNames).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(columnNames).To(HaveLen(len(columns) + reservedColumns))
}
