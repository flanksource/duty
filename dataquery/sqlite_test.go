package dataquery

import (
	"github.com/glebarez/sqlite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func testResultset(ctx context.Context, resultset QueryResultSet) {
	Expect(resultset.CreateDBTable(ctx)).To(Succeed())
	Expect(resultset.InsertToDB(ctx)).To(Succeed())

	var results []map[string]any
	Expect(ctx.DB().Table(resultset.Name).Find(&results).Error).To(Succeed())
	Expect(results).To(HaveLen(len(resultset.Results)))
}

var _ = Describe("Insert complex values", func() {
	var sqliteCtx = context.New()

	BeforeEach(func() {
		sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		Expect(err).ToNot(HaveOccurred())

		sqliteCtx = sqliteCtx.WithDB(sqliteDB, nil)
	})

	It("should insert columns with string map values", func() {
		resultSet := QueryResultSet{
			Name: "table1",
			Results: []QueryResultRow{
				{"id": 1, "name": map[string]string{"first": "alice", "last": "smith"}},
				{"id": 2, "name": map[string]string{"first": "bob", "last": "jones"}},
			},
		}

		testResultset(sqliteCtx, resultSet)
	})

	It("should insert columns with map values", func() {
		resultSet := QueryResultSet{
			Name: "table1",
			Results: []QueryResultRow{
				{"id": 1, "address": map[string]any{"street": map[string]any{"name": "123 Main St", "number": 123}, "city": "Othertown", "state": "NY", "zip": "67890"}},
				{"id": 2, "address": map[string]any{"street": map[string]any{"name": "456 Elm St", "number": 456}, "city": "Othertown", "state": "NY", "zip": "67890"}},
			},
		}

		testResultset(sqliteCtx, resultSet)

		var streetNames []string
		Expect(sqliteCtx.DB().Table(resultSet.Name).Select(`address->"street"->>"name"`).Find(&streetNames).Error).To(Succeed())
		Expect(streetNames).To(ConsistOf("123 Main St", "456 Elm St"))
	})

	It("should insert columns with struct values", func() {
		type Street struct {
			Name   string `json:"name"`
			Number int    `json:"number"`
		}

		type Address struct {
			Street Street `json:"street"`
			City   string `json:"city"`
		}

		resultSet := QueryResultSet{
			Name: "table1",
			Results: []QueryResultRow{
				{"id": 1, "address": Address{Street: Street{Name: "123 Main St", Number: 123}, City: "Othertown"}},
				{"id": 2, "address": Address{Street: Street{Name: "456 Elm St", Number: 456}, City: "Othertown"}},
			},
		}

		testResultset(sqliteCtx, resultSet)

		var streetNames []string
		Expect(sqliteCtx.DB().Table(resultSet.Name).Select(`address->"street"->>"name"`).Find(&streetNames).Error).To(Succeed())
		Expect(streetNames).To(ConsistOf("123 Main St", "456 Elm St"))
	})
})

var _ = Describe("InferColumnTypes", func() {
	It("should infer column types correctly", func() {
		rows := []QueryResultRow{
			{"id": 1, "name": "test1", "score": 95.5, "active": true},
			{"id": 2, "name": "test2", "score": 87.2, "active": false},
			{"id": 3, "name": "test3", "score": 92.1, "active": true, "details": map[string]any{"cluster": "test"}},
			{"id": 4, "name": "test4", "score": 90.1, "active": true, "tags": "cluster=test"},
		}

		columnTypes := InferColumnTypes(rows)

		Expect(columnTypes).To(HaveLen(6))
		Expect(columnTypes["id"]).To(Equal("INTEGER"))
		Expect(columnTypes["name"]).To(Equal("TEXT"))
		Expect(columnTypes["score"]).To(Equal("REAL"))
		Expect(columnTypes["active"]).To(Equal("INTEGER"))
		Expect(columnTypes["tags"]).To(Equal("TEXT"))
		Expect(columnTypes["details"]).To(Equal("BLOB"))
	})

	It("should handle mixed types", func() {
		rows := []QueryResultRow{
			{"id": 1, "score": 95.5},
			{"id": 2, "score": "90.1"},
		}

		columnTypes := InferColumnTypes(rows)

		Expect(columnTypes).To(HaveLen(2))
		Expect(columnTypes["id"]).To(Equal("INTEGER"))
		Expect(columnTypes["score"]).To(Equal("TEXT"))
	})

	It("should handle empty rows", func() {
		rows := []QueryResultRow{}
		columnTypes := InferColumnTypes(rows)
		Expect(columnTypes).To(HaveLen(0))
	})
})

var _ = Describe("Empty results with ColumnDefs", func() {
	var sqliteCtx = context.New()

	BeforeEach(func() {
		sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		Expect(err).ToNot(HaveOccurred())

		sqliteCtx = sqliteCtx.WithDB(sqliteDB, nil)
	})

	It("should create table from empty results using ColumnDefs", func() {
		resultSet := QueryResultSet{
			Name:    "prometheus_metrics",
			Results: []QueryResultRow{}, // Empty results
			ColumnDefs: map[string]models.ColumnType{
				"value":     models.ColumnTypeDecimal,
				"timestamp": models.ColumnTypeDateTime,
				"pod":       models.ColumnTypeString,
				"namespace": models.ColumnTypeString,
			},
		}

		// Should succeed in creating table despite empty results
		Expect(resultSet.CreateDBTable(sqliteCtx)).To(Succeed())
		Expect(resultSet.InsertToDB(sqliteCtx)).To(Succeed())

		// Verify table was created with correct schema
		var tableCount int64
		err := sqliteCtx.DB().Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", resultSet.Name).Scan(&tableCount).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(tableCount).To(Equal(int64(1)))

		// Verify table is empty but queryable
		var results []map[string]any
		Expect(sqliteCtx.DB().Table(resultSet.Name).Find(&results).Error).To(Succeed())
		Expect(results).To(HaveLen(0))
	})

	It("should fail when empty results have no ColumnDefs", func() {
		resultSet := QueryResultSet{
			Name:       "empty_table",
			Results:    []QueryResultRow{}, // Empty results
			ColumnDefs: nil,                // No column definitions
		}

		// Should fail to create table
		err := resultSet.CreateDBTable(sqliteCtx)
		Expect(err).To(HaveOccurred())
	})
})
