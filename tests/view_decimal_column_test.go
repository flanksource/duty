package tests

import (
	"fmt"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pkgView "github.com/flanksource/duty/view"
)

// A decimal column has to be given a numeric Postgres type. Falling back to text gives a
// column that pgx cannot encode a float into, so the whole view refresh fails on the first
// fractional value rather than on anything the view spec did wrong.
var _ = ginkgo.Describe("View decimal columns", ginkgo.Serial, func() {
	const tableName = "view_default_decimal_column_test"

	columns := pkgView.ViewColumnDefList{
		{Name: "id", Type: pkgView.ColumnTypeString, PrimaryKey: true},
		{Name: "cost", Type: pkgView.ColumnTypeDecimal},
	}

	ginkgo.AfterEach(func() {
		Expect(DefaultContext.DB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Error).To(Succeed())
	})

	ginkgo.It("gives a decimal column a numeric type", func() {
		Expect(pkgView.CreateViewTable(DefaultContext, tableName, columns)).To(Succeed())

		var dataType string
		Expect(DefaultContext.DB().Raw(`
			SELECT data_type FROM information_schema.columns
			WHERE table_name = ? AND column_name = 'cost'`, tableName).
			Scan(&dataType).Error).To(Succeed())

		Expect(dataType).To(Equal("numeric"))
	})

	// The upgrade path for a view table already created while decimal fell through to
	// text. Postgres will not cast text to numeric on its own, so the ALTER fails and the
	// table is dropped and rebuilt — which costs nothing here, the table being a cache the
	// next refresh repopulates.
	ginkgo.It("migrates a column already created as text", func() {
		asText := pkgView.ViewColumnDefList{
			{Name: "id", Type: pkgView.ColumnTypeString, PrimaryKey: true},
			{Name: "cost", Type: pkgView.ColumnTypeString},
		}
		Expect(pkgView.CreateViewTable(DefaultContext, tableName, asText)).To(Succeed())
		Expect(pkgView.CreateViewTable(DefaultContext, tableName, columns)).To(Succeed())

		var dataType string
		Expect(DefaultContext.DB().Raw(`
			SELECT data_type FROM information_schema.columns
			WHERE table_name = ? AND column_name = 'cost'`, tableName).
			Scan(&dataType).Error).To(Succeed())
		Expect(dataType).To(Equal("numeric"))

		rows := []pkgView.Row{{"resource-1", 214.8}}
		Expect(pkgView.InsertViewRows(DefaultContext, tableName, columns, rows, "")).To(Succeed())
	})

	ginkgo.It("stores a fractional value without an encoding error", func() {
		Expect(pkgView.CreateViewTable(DefaultContext, tableName, columns)).To(Succeed())

		rows := []pkgView.Row{{"resource-1", 214.8}}
		Expect(pkgView.InsertViewRows(DefaultContext, tableName, columns, rows, "")).To(Succeed())

		var stored float64
		Expect(DefaultContext.DB().Raw(fmt.Sprintf("SELECT cost FROM %s WHERE id = ?", tableName), "resource-1").
			Scan(&stored).Error).To(Succeed())

		Expect(stored).To(BeNumerically("==", 214.8))
	})
})
