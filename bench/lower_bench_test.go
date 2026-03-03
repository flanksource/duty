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
	label       string
	sqlNoLower  string
	argsNoLower []any
	sqlLower    string
	argsLower   []any
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

func printReport(t testing.TB, title string, withoutLower, withLower []queryPlan) {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 4, 3, ' ', 0)
	fmt.Fprintf(w, "\n%s\n\n", title)
	fmt.Fprintf(w, "Query\tScan (no LOWER)\tIndex\tTime\tScan (LOWER)\tIndex\tTime\n")
	fmt.Fprintf(w, "-----\t---------------\t-----\t----\t------------\t-----\t----\n")
	for i := range withoutLower {
		wo := withoutLower[i]
		wi := withLower[i]
		idx := wo.indexName
		if idx == "" {
			idx = "-"
		}
		idxL := wi.indexName
		if idxL == "" {
			idxL = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			wo.label, wo.scanType, idx, wo.execTime,
			wi.scanType, idxL, wi.execTime)
	}
	w.Flush()
	t.Log(buf.String())
}

func collectReports(b *testing.B, queries []queryDef, title string) {
	var withoutPlans, withPlans []queryPlan
	for _, q := range queries {
		withoutPlans = append(withoutPlans, collectPlan(q.label, q.sqlNoLower, q.argsNoLower...))
		withPlans = append(withPlans, collectPlan(q.label, q.sqlLower, q.argsLower...))
	}
	printReport(b, title, withoutPlans, withPlans)
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
	sampleNameLower := strings.ToLower(sampleName)

	queries := []queryDef{
		{
			label:       "exact_name",
			sqlNoLower:  "SELECT id FROM config_items WHERE name = $1",
			argsNoLower: []any{sampleName},
			sqlLower:    "SELECT id FROM config_items WHERE LOWER(CAST(name AS TEXT)) = $1",
			argsLower:   []any{sampleNameLower},
		},
		{
			label:       "exact_type",
			sqlNoLower:  "SELECT id FROM config_items WHERE type = $1",
			argsNoLower: []any{"Kubernetes::Pod"},
			sqlLower:    "SELECT id FROM config_items WHERE LOWER(CAST(type AS TEXT)) = $1",
			argsLower:   []any{"kubernetes::pod"},
		},
		{
			label:       "name_and_type",
			sqlNoLower:  "SELECT id FROM config_items WHERE name = $1 AND type = $2",
			argsNoLower: []any{sampleName, "Kubernetes::Pod"},
			sqlLower:    "SELECT id FROM config_items WHERE LOWER(CAST(name AS TEXT)) = $1 AND LOWER(CAST(type AS TEXT)) = $2",
			argsLower:   []any{sampleNameLower, "kubernetes::pod"},
		},
		{
			label:       "type_prefix",
			sqlNoLower:  "SELECT id FROM config_items WHERE type LIKE $1",
			argsNoLower: []any{"Kubernetes%"},
			sqlLower:    "SELECT id FROM config_items WHERE LOWER(CAST(type AS TEXT)) LIKE $1",
			argsLower:   []any{"kubernetes%"},
		},
	}

	reportTitle := fmt.Sprintf("=== INDEX USAGE REPORT (%dk config_items) ===", count/1000)
	collectReports(b, queries, reportTitle)

	for _, q := range queries {
		q := q
		b.Run(q.label, func(b *testing.B) {
			b.Run("Without_LOWER", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var results []string
					testCtx.DB().Raw(q.sqlNoLower, q.argsNoLower...).Pluck("id", &results)
				}
			})
			b.Run("With_LOWER", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var results []string
					testCtx.DB().Raw(q.sqlLower, q.argsLower...).Pluck("id", &results)
				}
			})
		})
	}
}
