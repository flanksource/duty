package bench_test

import (
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"
)

const lowerBenchSeedCount = 1_000_000

var lowerBenchTypes = []string{
	"Kubernetes::Pod",
	"Kubernetes::Node",
	"Kubernetes::Deployment",
	"Kubernetes::Service",
	"Kubernetes::ConfigMap",
	"Kubernetes::Secret",
	"Kubernetes::Ingress",
	"Kubernetes::StatefulSet",
	"Kubernetes::DaemonSet",
	"Kubernetes::Job",
	"Kubernetes::CronJob",
	"Kubernetes::ReplicaSet",
	"Kubernetes::Namespace",
	"Kubernetes::PersistentVolume",
	"Kubernetes::ServiceAccount",
	"AWS::EC2::Instance",
	"AWS::S3::Bucket",
	"AWS::RDS::DBInstance",
	"AWS::IAM::Role",
	"AWS::Lambda::Function",
}

type queryDef struct {
	label     string
	sql       string // plain query
	sqlLower  string // LOWER() wrapped query
	argsExact []any  // original case args
	argsLower []any  // lowercase args
}

type queryPlan struct {
	label     string
	scanType  string
	indexName string
	execTime  string
	rows      string
}

func seedLowerBenchData(t testing.TB) {
	quoted := make([]string, len(lowerBenchTypes))
	for i, item := range lowerBenchTypes {
		quoted[i] = fmt.Sprintf("'%s'", item)
	}
	typesArray := "(ARRAY[" + strings.Join(quoted, ",") + "])"

	sql := fmt.Sprintf(`
		INSERT INTO config_items (name, type, config_class)
		SELECT
			'bench-' || substr(md5(random()::text), 1, 20) || '-' || i,
			%s[1 + (i %% %d)],
			%s[1 + (i %% %d)]
		FROM generate_series(1, %d) AS i
	`, typesArray, len(lowerBenchTypes), typesArray, len(lowerBenchTypes), lowerBenchSeedCount)

	if err := testCtx.DB().Exec(sql).Error; err != nil {
		t.Fatalf("failed to seed benchmark data: %v", err)
	}

	if err := testCtx.DB().Exec("ANALYZE config_items").Error; err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}
}

func cleanupLowerBenchData(t testing.TB) {
	testCtx.DB().Exec("DELETE FROM config_items WHERE name LIKE 'bench-%'")
	testCtx.DB().Exec("ANALYZE config_items")
}

func parsePlan(plans []string) queryPlan {
	var qp queryPlan
	for _, line := range plans {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Index Scan") || strings.HasPrefix(trimmed, "Index Only Scan"):
			qp.scanType = "Index Scan"
			if idx := strings.Index(trimmed, "using "); idx != -1 {
				rest := trimmed[idx+6:]
				if end := strings.IndexAny(rest, " "); end != -1 {
					qp.indexName = rest[:end]
				}
			}
		case strings.HasPrefix(trimmed, "Bitmap"):
			if qp.scanType == "" {
				qp.scanType = "Bitmap Scan"
			}
			if idx := strings.Index(trimmed, "on "); idx != -1 {
				rest := trimmed[idx+3:]
				if end := strings.IndexAny(rest, " "); end != -1 {
					qp.indexName = rest[:end]
				}
			}
		case strings.HasPrefix(trimmed, "Seq Scan") || strings.Contains(trimmed, "Parallel Seq Scan"):
			if qp.scanType == "" {
				qp.scanType = "Seq Scan"
			}
		}

		if strings.Contains(trimmed, "Execution Time:") {
			qp.execTime = strings.TrimPrefix(trimmed, "Execution Time: ")
		}
		if strings.Contains(trimmed, "actual time=") {
			if idx := strings.Index(trimmed, "rows="); idx != -1 {
				rest := trimmed[idx+5:]
				if end := strings.IndexAny(rest, " "); end != -1 {
					qp.rows = rest[:end]
				}
			}
		}
	}
	return qp
}

func collectPlan(label, sql string, args ...any) queryPlan {
	var plans []string
	testCtx.DB().Raw("EXPLAIN (ANALYZE, BUFFERS) "+sql, args...).Pluck("QUERY PLAN", &plans)
	qp := parsePlan(plans)
	qp.label = label
	return qp
}

func printReport(t testing.TB, title, col1Label string, col1Plans []queryPlan, col2Label string, col2Plans []queryPlan) {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 4, 3, ' ', 0)
	fmt.Fprintf(w, "\n%s\n\n", title)
	fmt.Fprintf(w, "Query\tScan (%s)\tIndex\tTime\tScan (%s)\tIndex\tTime\n", col1Label, col2Label)
	fmt.Fprintf(w, "-----\t%s\t-----\t----\t%s\t-----\t----\n",
		strings.Repeat("-", len("Scan (")+len(col1Label)+1),
		strings.Repeat("-", len("Scan (")+len(col2Label)+1))
	for i := range col1Plans {
		p1 := col1Plans[i]
		p2 := col2Plans[i]
		idx1, idx2 := p1.indexName, p2.indexName
		if idx1 == "" {
			idx1 = "-"
		}
		if idx2 == "" {
			idx2 = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			p1.label, p1.scanType, idx1, p1.execTime,
			p2.scanType, idx2, p2.execTime)
	}
	w.Flush()
	t.Log(buf.String())
}

func buildQueries(sampleName string) []queryDef {
	sampleNameLower := strings.ToLower(sampleName)
	return []queryDef{
		{
			label:     "exact_name",
			sql:       "SELECT id FROM config_items WHERE name = $1",
			sqlLower:  "SELECT id FROM config_items WHERE LOWER(name) = $1",
			argsExact: []any{sampleName},
			argsLower: []any{sampleNameLower},
		},
		{
			label:     "exact_type",
			sql:       "SELECT id FROM config_items WHERE type = $1",
			sqlLower:  "SELECT id FROM config_items WHERE LOWER(type) = $1",
			argsExact: []any{"Kubernetes::Pod"},
			argsLower: []any{"kubernetes::pod"},
		},
		{
			label:     "name_and_type",
			sql:       "SELECT id FROM config_items WHERE name = $1 AND type = $2",
			sqlLower:  "SELECT id FROM config_items WHERE LOWER(name) = $1 AND LOWER(type) = $2",
			argsExact: []any{sampleName, "Kubernetes::Pod"},
			argsLower: []any{sampleNameLower, "kubernetes::pod"},
		},
		{
			label:     "type_prefix",
			sql:       "SELECT id FROM config_items WHERE type LIKE $1",
			sqlLower:  "SELECT id FROM config_items WHERE LOWER(type) LIKE $1",
			argsExact: []any{"Kubernetes%"},
			argsLower: []any{"kubernetes%"},
		},
	}
}

func createLowerIndexes(b *testing.B) {
	indexes := []string{
		"CREATE INDEX idx_config_items_lower_name ON config_items (lower(name))",
		"CREATE INDEX idx_config_items_lower_type ON config_items (lower(type))",
	}
	for _, ddl := range indexes {
		if err := testCtx.DB().Exec(ddl).Error; err != nil {
			b.Fatalf("failed to create index: %v", err)
		}
	}
	if err := testCtx.DB().Exec("ANALYZE config_items").Error; err != nil {
		b.Fatalf("failed to analyze after index creation: %v", err)
	}
}

func dropLowerIndexes(b *testing.B) {
	for _, idx := range []string{"idx_config_items_lower_name", "idx_config_items_lower_type"} {
		testCtx.DB().Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", idx))
	}
}

func runPhase(b *testing.B, queries []queryDef, title string) {
	var exactPlans, lowerPlans []queryPlan
	for _, q := range queries {
		exactPlans = append(exactPlans, collectPlan(q.label, q.sql, q.argsExact...))
		lowerPlans = append(lowerPlans, collectPlan(q.label, q.sqlLower, q.argsLower...))
	}
	printReport(b, title, "Exact", exactPlans, "LOWER()", lowerPlans)

	for _, q := range queries {
		q := q
		b.Run(q.label, func(b *testing.B) {
			b.Run("Exact_Match", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var results []string
					testCtx.DB().Raw(q.sql, q.argsExact...).Pluck("id", &results)
				}
			})
			b.Run("Case_Insensitive", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var results []string
					testCtx.DB().Raw(q.sqlLower, q.argsLower...).Pluck("id", &results)
				}
			})
		})
	}
}

func BenchmarkLowerCase(b *testing.B) {
	resetPG(b, false)
	seedLowerBenchData(b)
	defer cleanupLowerBenchData(b)

	var count int64
	testCtx.DB().Raw("SELECT COUNT(*) FROM config_items").Scan(&count)
	if count < int64(lowerBenchSeedCount) {
		b.Fatalf("expected at least %d config_items, got %d", lowerBenchSeedCount, count)
	}

	var sampleName string
	testCtx.DB().Raw("SELECT name FROM config_items WHERE name LIKE 'bench-%' LIMIT 1").Scan(&sampleName)

	queries := buildQueries(sampleName)

	b.Run("Without_LOWER_Index", func(b *testing.B) {
		dropLowerIndexes(b)
		runPhase(b, queries, fmt.Sprintf("=== WITHOUT LOWER INDEX (%dk rows) ===", count/1000))
	})

	b.Run("With_LOWER_Index", func(b *testing.B) {
		createLowerIndexes(b)
		defer dropLowerIndexes(b)
		runPhase(b, queries, fmt.Sprintf("=== WITH LOWER INDEX (%dk rows) ===", count/1000))
	})
}
