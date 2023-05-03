package config

import (
	"testing"
)

func TestValidateTablesInQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		allowedPrefix  []string
		expectedOutput bool
	}{
		// Simple SELECT queries
		{"Simple select with allowed table", "SELECT * FROM config_items", []string{"config_"}, true},
		{"Simple select with disallowed table", "SELECT * FROM users", []string{"config_"}, false},
		{"Simple select with allowed table 2", "SELECT * FROM users", []string{"users"}, true},

		// Joins
		{"Join with allowed tables", "SELECT * FROM config_items JOIN config_changes ON config_items.id = config_changes.config_id", []string{"config_"}, true},
		{"Left join with allowed tables", "SELECT * FROM config_items LEFT JOIN config_analysis ON config_items.id = config_analysis.config_id", []string{"config_"}, true},
		{"Left join with multiple allowed tables", "SELECT * FROM evidences LEFT JOIN config_items ON evidences.config_id = config_items.id", []string{"config_", "evidences"}, true},
		{"Left join with only one allowed table", "SELECT * FROM evidences LEFT JOIN config_items ON evidences.config_id = config_items.id", []string{"config_"}, false},
		{"Left join with multiple allowed tables 2", "SELECT * FROM config_items LEFT JOIN evidences ON evidences.config_id = config_items.id", []string{"config_", "evidences"}, true},
		{"Left join with only one allowed table 2", "SELECT * FROM config_items LEFT JOIN evidences ON evidences.config_id = config_items.id", []string{"config_"}, false},

		// Unions
		{"Union with disallowed table", "SELECT * FROM users UNION SELECT * FROM config_items", []string{"config_"}, false},
		{"Union with allowed table", "SELECT id FROM config_analysis UNION SELECT id FROM config_items", []string{"config_"}, true},

		// Subqueries
		{"Subquery with allowed tables", "SELECT * FROM config_items WHERE id IN (SELECT config_id FROM config_changes)", []string{"config_"}, true},
		{"Subquery with disallowed table", "SELECT * FROM config_items WHERE id IN (SELECT id FROM people)", []string{"config_"}, false},
		{"Subquery with allowed tables 2", "SELECT * FROM config_items WHERE id IN (SELECT config_id FROM config_analysis)", []string{"config_"}, true},

		// CTEs
		// The sqlparser library doesn't seem to support CTE.
		// Returns "failed to parse SQL query: syntax error at position 5 near 'with'" error.
		// {"CTE with allowed tables", "WITH recent_changes AS (SELECT * FROM config_changes WHERE created_at > NOW() - INTERVAL '1 day') SELECT * FROM config_items JOIN recent_changes ON config_items.id = recent_changes.config_id", []string{"config_"}, true},
		// {"CTE with disallowed tables", "WITH recent_changes AS (SELECT * FROM config_changes WHERE created_at > NOW() - INTERVAL '1 day') SELECT * FROM config_items JOIN recent_changes ON config_items.id = recent_changes.config_id", []string{"config_items"}, false},

		// Nested
		{"Nested subquery with allowed tables", "SELECT * FROM config_items WHERE category_id IN (SELECT id FROM config_categories WHERE parent_id IN (SELECT id FROM config_categories WHERE name = 'Electronics'))", []string{"config_"}, true},
		{"Nested subquery with disallowed tables", "SELECT * FROM config_items WHERE category_id IN (SELECT id FROM config_categories WHERE parent_id IN (SELECT id FROM master_config WHERE name = 'Electronics'))", []string{"config_"}, false},

		// Aggregation
		{"Aggregation with GROUP BY and allowed tables", "SELECT category_id, COUNT(*) as item_count, AVG(value) as average_value FROM config_items GROUP BY category_id", []string{"config_"}, true},
		{"Aggregation with GROUP BY and allowed tables", "SELECT category_id, COUNT(*) as item_count, AVG(value) as average_value FROM config_items GROUP BY category_id", []string{"components_"}, false},

		// HAVING
		{"HAVING clause with allowed tables", "SELECT category_id, COUNT(*) as item_count FROM config_items GROUP BY category_id HAVING COUNT(*) > 10", []string{"config_"}, true},
		{"HAVING clause with allowed tables", "SELECT category_id, COUNT(*) as item_count FROM config_items GROUP BY category_id HAVING COUNT(*) > 10", []string{"components_"}, false},

		// Pivot
		{"Pivot operation with allowed tables", "SELECT category_id, SUM(CASE WHEN value_type = 'A' THEN value ELSE 0 END) as A_value, SUM(CASE WHEN value_type = 'B' THEN value ELSE 0 END) as B_value FROM config_items GROUP BY category_id", []string{"config_"}, true},
		{"Pivot operation with disallowed tables", "SELECT category_id, SUM(CASE WHEN value_type = 'A' THEN value ELSE 0 END) as A_value, SUM(CASE WHEN value_type = 'B' THEN value ELSE 0 END) as B_value FROM nonconfig_items GROUP BY category_id", []string{"config_"}, false},

		// Conditional
		{"Conditional expression with allowed tables", "SELECT id, name, value, CASE WHEN value >= 100 THEN 'High' ELSE 'Low' END as value_category FROM config_items", []string{"config_"}, true},
		{"Conditional expression with disallowed tables", "SELECT id, name, value, CASE WHEN value >= 100 THEN 'High' ELSE 'Low' END as value_category FROM config_items", []string{"components_"}, false},

		// Complex expressions
		{"Complex expressions in SELECT with allowed tables", "SELECT id, CONCAT(name, ' - ', category_id) as full_name, value * 1.1 as adjusted_value FROM config_items", []string{"config_"}, true},
		{"Complex expressions in SELECT with disallowed tables", "SELECT id, CONCAT(name, ' - ', category_id) as full_name, value * 1.1 as adjusted_value FROM config_items", []string{"components_"}, false},

		// Aliases
		{"Alias restricted table to allowed table", "SELECT config_users.* FROM (SELECT * FROM users) AS config_users", []string{"config_"}, false},
		{"Alias allowed table to allowed table", "SELECT configs.* FROM (SELECT * FROM config_items) AS configs", []string{"config_"}, true},
	}

	for _, test := range tests {
		output, err := validateTablesInQuery(test.query, test.allowedPrefix...)
		if err != nil {
			t.Logf("%s: validateTablesInQuery() returned an unexpected error: %v", test.name, err)
			t.Fail()
		}
		if output != test.expectedOutput {
			t.Logf("%s: validateTablesInQuery() returned %v, but expected %v", test.name, output, test.expectedOutput)
			t.Fail()
		}
	}
}

// var _ = ginkgo.Describe("Query should only run non mutation queries", func() {
// 	ginkgo.It("should support read query", func() {
// 		_, err := query(context.TODO(), testutils.TestDBPGPool, "SELECT id, created_at FROM config_items")
// 		Expect(err).To(Not(BeNil()))
// 	})

// 	ginkgo.It("should not support INSERT query", func() {
// 		_, err := query(context.TODO(), testutils.TestDBPGPool, "INSERT INTO config_changes(config_id, external_change_id) VALUES('0186a12e-b10d-befa-72ce-2a61f69e5ccd', 'whatever')")
// 		assertPreventCommandIfReadOnlyErr(err)
// 	})

// 	ginkgo.It("should not support UPDATE query", func() {
// 		_, err := query(context.TODO(), testutils.TestDBPGPool, "UPDATE config_changes SET external_change_id = '0186a12e-b10d-befa-72ce-2a61f69e5ccd' WHERE config_id = '0186a12e-b10d-befa-72ce-2a61f69e5ccd'")
// 		assertPreventCommandIfReadOnlyErr(err)
// 	})

// 	ginkgo.It("should not support DELETE query", func() {
// 		_, err := query(context.TODO(), testutils.TestDBPGPool, "DELETE FROM config_changes WHERE config_id ='0186a12e-b10d-befa-72ce-2a61f69e5ccd'")
// 		assertPreventCommandIfReadOnlyErr(err)
// 	})
// })

// func assertPreventCommandIfReadOnlyErr(err error) {
// 	Expect(err).ToNot(BeNil())

// 	var pgErr *pgconn.PgError
// 	Expect(errors.As(err, &pgErr)).To(BeTrue())

// 	Expect(pgErr.Code).To(Equal("25006")) //PreventCommandIfReadOnly
// }
