package dataquery

import (
	"testing"

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

func TestK8sCPUToNumber(t *testing.T) {
	g := NewWithT(t)

	g.Expect(k8sCPUToNumber("500m")).To(Equal(0.5))
	g.Expect(k8sCPUToNumber("1")).To(Equal(1.0))
	g.Expect(k8sCPUToNumber("2000m")).To(Equal(2.0))
	g.Expect(k8sCPUToNumber("1.5")).To(Equal(1.5))
	g.Expect(k8sCPUToNumber("")).To(Equal(0.0))
	g.Expect(k8sCPUToNumber("invalid")).To(Equal(0.0))
}

func TestK8sCPUToNumberSQL(t *testing.T) {
	g := NewWithT(t)

	resultset := QueryResultSet{
		Name: "cpu_test",
		Results: []QueryResultRow{
			{"id": 1, "cpu": "500m"},
			{"id": 2, "cpu": "1"},
			{"id": 3, "cpu": "2000m"},
			{"id": 4, "cpu": "1.5"},
			{"id": 5, "cpu": ""},
			{"id": 6, "cpu": "invalid"},
		},
	}

	ctx, cleanup, err := DBFromResultsets(context.New(), []QueryResultSet{resultset})
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = cleanup() }()

	var results []struct {
		ID      int     `gorm:"column:id"`
		CPUText string  `gorm:"column:cpu"`
		CPUNum  float64 `gorm:"column:cpu_num"`
	}

	err = ctx.DB().Table("cpu_test").
		Select("id, cpu, k8s_cpu_to_number(cpu) as cpu_num").
		Find(&results).Error
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(results).To(HaveLen(6))
	g.Expect(results[0].CPUNum).To(Equal(0.5)) // 500m -> 0.5
	g.Expect(results[1].CPUNum).To(Equal(1.0)) // 1 -> 1.0
	g.Expect(results[2].CPUNum).To(Equal(2.0)) // 2000m -> 2.0
	g.Expect(results[3].CPUNum).To(Equal(1.5)) // 1.5 -> 1.5
	g.Expect(results[4].CPUNum).To(Equal(0.0)) // empty -> 0.0
	g.Expect(results[5].CPUNum).To(Equal(0.0)) // invalid -> 0.0
}

func TestMemoryToBytes(t *testing.T) {
	g := NewWithT(t)

	g.Expect(memoryToBytes("500")).To(Equal(int64(500)))
	g.Expect(memoryToBytes("500KB")).To(Equal(int64(500000)))
	g.Expect(memoryToBytes("500MB")).To(Equal(int64(500000000)))
	g.Expect(memoryToBytes("1GB")).To(Equal(int64(1000000000)))
	g.Expect(memoryToBytes("2TB")).To(Equal(int64(2000000000000)))

	// Binary units
	g.Expect(memoryToBytes("1KiB")).To(Equal(int64(1024)))
	g.Expect(memoryToBytes("1MiB")).To(Equal(int64(1048576)))
	g.Expect(memoryToBytes("1GiB")).To(Equal(int64(1073741824)))
	g.Expect(memoryToBytes("1TiB")).To(Equal(int64(1099511627776)))

	// Short units
	g.Expect(memoryToBytes("500K")).To(Equal(int64(500000)))
	g.Expect(memoryToBytes("500M")).To(Equal(int64(500000000)))
	g.Expect(memoryToBytes("1G")).To(Equal(int64(1000000000)))
	g.Expect(memoryToBytes("2T")).To(Equal(int64(2000000000000)))

	// Case insensitive
	g.Expect(memoryToBytes("500kb")).To(Equal(int64(500000)))
	g.Expect(memoryToBytes("500mB")).To(Equal(int64(500000000)))
	g.Expect(memoryToBytes("500 MB")).To(Equal(int64(500000000)))

	// Edge cases
	g.Expect(memoryToBytes("")).To(Equal(int64(0)))
	g.Expect(memoryToBytes("invalid")).To(Equal(int64(0)))
	g.Expect(memoryToBytes("500XB")).To(Equal(int64(0)))
}

func TestMemoryToBytesSQL(t *testing.T) {
	g := NewWithT(t)

	resultset := QueryResultSet{
		Name: "memory_test",
		Results: []QueryResultRow{
			{"id": 1, "memory": "500"},
			{"id": 2, "memory": "500KB"},
			{"id": 3, "memory": "500MB"},
			{"id": 4, "memory": "1GB"},
			{"id": 5, "memory": "1KiB"},
			{"id": 6, "memory": "1MiB"},
			{"id": 7, "memory": "500K"},
			{"id": 8, "memory": "500M"},
			{"id": 9, "memory": "500kb"},
			{"id": 10, "memory": ""},
			{"id": 11, "memory": "invalid"},
		},
	}

	ctx, cleanup, err := DBFromResultsets(context.New(), []QueryResultSet{resultset})
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = cleanup() }()

	var results []struct {
		ID         int    `gorm:"column:id"`
		MemoryText string `gorm:"column:memory"`
		MemoryNum  int64  `gorm:"column:memory_num"`
	}

	err = ctx.DB().Table("memory_test").
		Select("id, memory, memory_to_bytes(memory) as memory_num").
		Find(&results).Error
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(results).To(HaveLen(11))
	g.Expect(results[0].MemoryNum).To(Equal(int64(500)))        // 500 -> 500
	g.Expect(results[1].MemoryNum).To(Equal(int64(500000)))     // 500KB -> 500000
	g.Expect(results[2].MemoryNum).To(Equal(int64(500000000)))  // 500MB -> 500000000
	g.Expect(results[3].MemoryNum).To(Equal(int64(1000000000))) // 1GB -> 1000000000
	g.Expect(results[4].MemoryNum).To(Equal(int64(1024)))       // 1KiB -> 1024
	g.Expect(results[5].MemoryNum).To(Equal(int64(1048576)))    // 1MiB -> 1048576
	g.Expect(results[6].MemoryNum).To(Equal(int64(500000)))     // 500K -> 500000
	g.Expect(results[7].MemoryNum).To(Equal(int64(500000000)))  // 500M -> 500000000
	g.Expect(results[8].MemoryNum).To(Equal(int64(500000)))     // 500kb -> 500000 (case insensitive)
	g.Expect(results[9].MemoryNum).To(Equal(int64(0)))          // empty -> 0
	g.Expect(results[10].MemoryNum).To(Equal(int64(0)))         // invalid -> 0
}

func TestSQLiteFunctionsWithIncorrectTypes(t *testing.T) {
	g := NewWithT(t)

	// Test data with mixed types
	resultset := QueryResultSet{
		Name: "mixed_types_test",
		Results: []QueryResultRow{
			{"id": 1, "memory_str": "500MB", "memory_num": 1024, "cpu_str": "500m", "cpu_num": 2.5},
			{"id": 2, "memory_str": "1GB", "memory_num": 2048, "cpu_str": "1", "cpu_num": 3.0},
		},
	}

	ctx, cleanup, err := DBFromResultsets(context.New(), []QueryResultSet{resultset})
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = cleanup() }()

	t.Run("memory_to_bytes with numeric input", func(t *testing.T) {
		g := NewWithT(t)

		var results []struct {
			ID        int   `gorm:"column:id"`
			MemoryNum int64 `gorm:"column:memory_result"`
		}

		// This should handle numeric input - SQLite will convert number to string
		err = ctx.DB().Table("mixed_types_test").
			Select("id, memory_to_bytes(memory_num) as memory_result").
			Find(&results).Error

		// The function expects a string but gets a number - SQLite converts to string
		// 1024 becomes "1024" -> 1024 bytes, 2048 becomes "2048" -> 2048 bytes
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results).To(HaveLen(2))
		g.Expect(results[0].MemoryNum).To(Equal(int64(1024))) // 1024 -> "1024" -> 1024 bytes
		g.Expect(results[1].MemoryNum).To(Equal(int64(2048))) // 2048 -> "2048" -> 2048 bytes
	})

	t.Run("k8s_cpu_to_number with numeric input", func(t *testing.T) {
		g := NewWithT(t)

		var results []struct {
			ID     int     `gorm:"column:id"`
			CPUNum float64 `gorm:"column:cpu_result"`
		}

		// This should handle numeric input - SQLite will convert number to string
		err = ctx.DB().Table("mixed_types_test").
			Select("id, k8s_cpu_to_number(cpu_num) as cpu_result").
			Find(&results).Error

		// The function expects a string but gets a number
		// SQLite converts 2.5 -> "2.5", 3.0 -> "3.0"
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results).To(HaveLen(2))
		g.Expect(results[0].CPUNum).To(Equal(2.5)) // 2.5 -> "2.5" -> 2.5
		g.Expect(results[1].CPUNum).To(Equal(3.0)) // 3.0 -> "3.0" -> 3.0
	})

	t.Run("functions with literal numeric values", func(t *testing.T) {
		g := NewWithT(t)

		var results []struct {
			MemoryNum int64   `gorm:"column:memory_result"`
			CPUNum    float64 `gorm:"column:cpu_result"`
		}

		// Test with literal numeric values passed to functions
		err = ctx.DB().Table("mixed_types_test").
			Select("memory_to_bytes(1024) as memory_result, k8s_cpu_to_number(2.5) as cpu_result").
			Limit(1).
			Find(&results).Error

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results).To(HaveLen(1))
		g.Expect(results[0].MemoryNum).To(Equal(int64(1024))) // 1024 -> "1024" -> 1024 bytes
		g.Expect(results[0].CPUNum).To(Equal(2.5))            // 2.5 -> "2.5" -> 2.5
	})
}
