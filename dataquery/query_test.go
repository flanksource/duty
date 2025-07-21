package dataquery

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
)

func TestDataQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DataQuery Suite")
}

var _ = Describe("MergeQueryResults", func() {
	Describe("UNION operations", func() {
		It("should merge result sets using UNION", func() {
			resultSet1 := QueryResultSet{
				Name: "table1",
				Results: []QueryResultRow{
					{"id": 1, "name": "alice"},
					{"id": 2, "name": "bob"},
				},
			}

			resultSet2 := QueryResultSet{
				Name: "table2",
				Results: []QueryResultRow{
					{"id": 3, "name": "charlie"},
					{"id": 4, "name": "diana"},
				},
			}

			ctx, closer, err := DBFromResultsets(context.New(), []QueryResultSet{resultSet1, resultSet2})
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := closer()
				Expect(err).ToNot(HaveOccurred())
			}()

			mergeQuery := "SELECT id, name FROM table1 UNION SELECT id, name FROM table2"
			results, err := RunSQL(ctx, mergeQuery)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(4))
			Expect(results).To(ConsistOf([]QueryResultRow{
				{"id": int64(1), "name": "alice"},
				{"id": int64(2), "name": "bob"},
				{"id": int64(3), "name": "charlie"},
				{"id": int64(4), "name": "diana"},
			}))
		})
	})

	Describe("JOIN operations", func() {
		It("should handle LEFT JOIN", func() {
			resultSet1 := QueryResultSet{
				Name: "users",
				Results: []QueryResultRow{
					{"id": 1, "name": "alice"},
					{"id": 2, "name": "bob"},
					{"id": 3, "name": "charlie"},
				},
			}

			resultSet2 := QueryResultSet{
				Name: "orders",
				Results: []QueryResultRow{
					{"user_id": 1, "product": "laptop"},
					{"user_id": 2, "product": "mouse"},
				},
			}

			mergeQuery := `SELECT 
				users.id AS "users.id",
				users.name AS "users.name",
				orders.user_id AS "orders.user_id",
				orders.product AS "orders.product"
			FROM users LEFT JOIN orders ON users.id = orders.user_id`

			ctx, closer, err := DBFromResultsets(context.New(), []QueryResultSet{resultSet1, resultSet2})
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := closer()
				Expect(err).ToNot(HaveOccurred())
			}()

			results, err := RunSQL(ctx, mergeQuery)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results).To(ConsistOf([]QueryResultRow{
				{"users.id": int64(1), "users.name": "alice", "orders.user_id": int64(1), "orders.product": "laptop"},
				{"users.id": int64(2), "users.name": "bob", "orders.user_id": int64(2), "orders.product": "mouse"},
				{"users.id": int64(3), "users.name": "charlie", "orders.user_id": nil, "orders.product": nil},
			}))
		})

		It("should handle JOIN with 3 tables", func() {
			resultSet1 := QueryResultSet{
				Name: "users",
				Results: []QueryResultRow{
					{"id": 1, "name": "alice", "department_id": 1},
					{"id": 2, "name": "bob", "department_id": 2},
					{"id": 3, "name": "charlie", "department_id": 1},
				},
			}

			resultSet2 := QueryResultSet{
				Name: "orders",
				Results: []QueryResultRow{
					{"user_id": 1, "product": "laptop", "order_id": 101},
					{"user_id": 2, "product": "mouse", "order_id": 102},
					{"user_id": 3, "product": "keyboard", "order_id": 103},
				},
			}

			resultSet3 := QueryResultSet{
				Name: "departments",
				Results: []QueryResultRow{
					{"id": 1, "name": "Engineering"},
					{"id": 2, "name": "Sales"},
				},
			}

			mergeQuery := `SELECT 
				users.id AS "users.id",
				users.name AS "users.name",
				users.department_id AS "users.department_id",
				orders.user_id AS "orders.user_id",
				orders.product AS "orders.product",
				orders.order_id AS "orders.order_id",
				departments.id AS "departments.id",
				departments.name AS "departments.name"
			FROM users 
			LEFT JOIN orders ON users.id = orders.user_id
			LEFT JOIN departments ON users.department_id = departments.id`

			ctx, closer, err := DBFromResultsets(context.New(), []QueryResultSet{resultSet1, resultSet2, resultSet3})
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := closer()
				Expect(err).ToNot(HaveOccurred())
			}()

			results, err := RunSQL(ctx, mergeQuery)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results).To(ConsistOf([]QueryResultRow{
				{
					"users.id": int64(1), "users.name": "alice", "users.department_id": int64(1),
					"orders.user_id": int64(1), "orders.product": "laptop", "orders.order_id": int64(101),
					"departments.id": int64(1), "departments.name": "Engineering",
				},
				{
					"users.id": int64(2), "users.name": "bob", "users.department_id": int64(2),
					"orders.user_id": int64(2), "orders.product": "mouse", "orders.order_id": int64(102),
					"departments.id": int64(2), "departments.name": "Sales",
				},
				{
					"users.id": int64(3), "users.name": "charlie", "users.department_id": int64(1),
					"orders.user_id": int64(3), "orders.product": "keyboard", "orders.order_id": int64(103),
					"departments.id": int64(1), "departments.name": "Engineering",
				},
			}))
		})
	})
})
