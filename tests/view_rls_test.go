package tests

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
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

var _ = Describe("Generated view table policy", Ordered, Serial, func() {
	columns := view.ViewColumnDefList{
		{Name: "id", Type: view.ColumnTypeString, PrimaryKey: true},
	}

	var (
		viewIDs  []uuid.UUID
		scopeIDs []uuid.UUID
		tables   []string
	)

	// newView stores a view, creates its generated table and fills it with rows.
	newView := func(namespace, name string, rows int) models.View {
		GinkgoHelper()
		v := models.View{ID: uuid.New(), Namespace: namespace, Name: name, Spec: types.JSON(`{}`)}
		Expect(DefaultContext.DB().Create(&v).Error).To(Succeed())
		viewIDs = append(viewIDs, v.ID)

		table := v.GeneratedTableName()
		tables = append(tables, table)
		Expect(view.CreateViewTable(DefaultContext, table, columns)).To(Succeed())
		for range rows {
			Expect(DefaultContext.DB().Exec(fmt.Sprintf("INSERT INTO %s (id) VALUES (?)", pq.QuoteIdentifier(table)), uuid.NewString()).Error).To(Succeed())
		}
		return v
	}

	// viewScope builds a Scope selecting the view.
	viewScope := func(v models.View) rls.Payload {
		GinkgoHelper()
		scope := models.Scope{ID: uuid.New(), Name: "view-table-" + v.ID.String(), Namespace: "rls-test", Targets: types.JSON(`[]`)}
		Expect(DefaultContext.DB().Create(&scope).Error).To(Succeed())
		_, err := membership.Rebuild(DefaultContext, scope.ID, scope.Targets, []membership.Target{
			{Type: policy.ResourceView, Selector: types.ResourceSelector{ID: v.ID.String()}},
		})
		Expect(err).ToNot(HaveOccurred())
		scopeIDs = append(scopeIDs, scope.ID)
		return rls.Payload{View: grants(rls.Grant{Scope: scope.ID.String()})}
	}

	// readable counts the table's rows each PostgREST role lists under the claim; both must list the same.
	readable := func(table string, payload rls.Payload) int64 {
		GinkgoHelper()
		var counts []int64
		for _, role := range []string{"postgrest_api", "postgrest_anon"} {
			tx := DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})
			Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(Succeed())
			counts = append(counts, countRows(tx, table, payload))
			Expect(tx.Rollback().Error).To(Succeed())
		}
		Expect(counts[0]).To(Equal(counts[1]), "postgrest_api and postgrest_anon list different rows")
		return counts[0]
	}

	policyOID := func(table string) uint32 {
		GinkgoHelper()
		var oid uint32
		Expect(DefaultContext.DB().Raw(`SELECT p.oid FROM pg_policy p JOIN pg_class c ON c.oid = p.polrelid
			WHERE c.relname = ?::name AND p.polname = 'view_grants_policy'`, table).Scan(&oid).Error).To(Succeed())
		return oid
	}

	AfterAll(func() {
		for _, table := range tables {
			Expect(DefaultContext.DB().Exec("DROP TABLE IF EXISTS " + pq.QuoteIdentifier(table)).Error).To(Succeed())
		}
		Expect(DefaultContext.DB().Exec("DELETE FROM scopes WHERE id IN ?", scopeIDs).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM views WHERE id IN ?", viewIDs).Error).To(Succeed())
	})

	It("lists every row of a granted view and none through another view's grant", func() {
		table := dummy.PodView.GeneratedTableName()
		Expect(countAll(table, "")).To(BeNumerically(">", 0))

		Expect(readable(table, viewScope(dummy.PodView))).To(Equal(countAll(table, "")))
		Expect(readable(table, viewScope(dummy.ViewDev))).To(BeZero())
		Expect(readable(table, rls.Payload{})).To(BeZero())
	})

	It("lists no row of a table two views share, through either view's grant", func() {
		teamA := newView("rls-team-a", "pods", 2)
		team := newView("rls-team", "a-pods", 1)
		Expect(teamA.GeneratedTableName()).To(Equal(team.GeneratedTableName()))

		table := teamA.GeneratedTableName()
		teamAGrant, teamGrant := viewScope(teamA), viewScope(team)
		Expect(readable(table, teamAGrant)).To(BeZero())
		Expect(readable(table, teamGrant)).To(BeZero())

		Expect(DefaultContext.DB().Model(&models.View{}).Where("id = ?", team.ID).Update("deleted_at", gorm.Expr("NOW()")).Error).To(Succeed())
		Expect(readable(table, teamAGrant)).To(BeZero(), "the deleted view's rows stay in the table")
	})

	It("lists the rows of a view re-applied over its deleted row", func() {
		old := newView("rls-reapplied", "pods", 2)
		Expect(DefaultContext.DB().Model(&models.View{}).Where("id = ?", old.ID).Update("deleted_at", gorm.Expr("NOW()")).Error).To(Succeed())
		current := newView("rls-reapplied", "pods", 0)

		Expect(readable(current.GeneratedTableName(), viewScope(current))).To(Equal(int64(2)))
		Expect(readable(current.GeneratedTableName(), viewScope(old))).To(BeZero())
	})

	It("resolves long multibyte names the way Postgres truncates them", func() {
		// Both names truncate to the same 61 bytes: the 3-byte character crosses the 63-byte limit.
		long := strings.Repeat("é", 26) + "日"
		first := newView("rls", long+"-first", 1)
		second := newView("rls", long+"-second", 1)
		Expect(len(first.GeneratedTableName())).To(BeNumerically(">", 63))

		table := first.GeneratedTableName()
		Expect(countAll(table, "")).To(Equal(int64(2)), "both views write to one table")
		Expect(readable(table, viewScope(first))).To(BeZero())
		Expect(readable(second.GeneratedTableName(), viewScope(second))).To(BeZero())

		solo := newView("rls-solo", long+"-solo", 3)
		Expect(readable(solo.GeneratedTableName(), viewScope(solo))).To(Equal(int64(3)))

		oid := policyOID(solo.GeneratedTableName())
		Expect(DefaultContext.DB().Exec("SELECT ensure_view_table_policy(?)", solo.GeneratedTableName()).Error).To(Succeed())
		Expect(policyOID(solo.GeneratedTableName())).To(Equal(oid), "a current policy isn't rewritten")
	})

	It("upgrades an older policy", func() {
		v := newView("rls-upgrade", "pods", 2)
		table := v.GeneratedTableName()
		grant := viewScope(v)

		for _, older := range []string{
			"FOR ALL TO postgrest_api, postgrest_anon USING (check_view_grants(__grants))",
			"FOR SELECT TO postgrest_api, postgrest_anon USING (check_view_grants(__grants))",
		} {
			Expect(DefaultContext.DB().Exec("DROP POLICY view_grants_policy ON " + table).Error).To(Succeed())
			Expect(DefaultContext.DB().Exec("CREATE POLICY view_grants_policy ON " + table + " " + older).Error).To(Succeed())
			Expect(readable(table, grant)).To(BeZero(), older)

			Expect(view.CreateViewTable(DefaultContext, table, columns)).To(Succeed())

			var cmd string
			Expect(DefaultContext.DB().Raw(`SELECT cmd FROM pg_policies WHERE tablename = ? AND policyname = 'view_grants_policy'
				AND qual LIKE '%view_table_readable%'`, table).Scan(&cmd).Error).To(Succeed())
			Expect(cmd).To(Equal("SELECT"), older)
			Expect(readable(table, grant)).To(Equal(int64(2)), older)
		}
	})

	It("refuses writes from PostgREST roles", func() {
		table := dummy.PodView.GeneratedTableName()
		before := countAll(table, "")

		for _, role := range []string{"postgrest_api", "postgrest_anon"} {
			for _, write := range []string{
				fmt.Sprintf("INSERT INTO %s (id) VALUES ('%s')", table, uuid.NewString()),
				fmt.Sprintf("UPDATE %s SET name = 'changed'", table),
				fmt.Sprintf("DELETE FROM %s", table),
			} {
				tx := DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin()
				Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(Succeed())
				Expect((rls.Payload{View: rls.AllRows()}).SetPostgresSessionRLS(tx)).To(Succeed())
				result := tx.Exec(write)
				if result.Error == nil {
					Expect(result.RowsAffected).To(BeZero(), role+": "+write)
				}
				Expect(tx.Rollback().Error).To(Succeed())
			}
		}

		Expect(countAll(table, "")).To(Equal(before))
		Expect(countAll(table, "name = 'changed'")).To(BeZero())
	})
})
