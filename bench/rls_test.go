package bench_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
	pkgRLS "github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/types"
)

// number of total configs in the database
var defaultTestSizes = []int{10_000, 25_000, 50_000, 100_000}

type DistinctBenchConfig struct {
	// view/table name
	relation string

	// optional column to fetch.
	// when left empty all columns are fetched (this is left empty for views with single column)
	column string
}

// views with `tags` column
// var viewsWithTags = []string{"catalog_changes", "config_detail", "configs"}

var benchConfigs = []DistinctBenchConfig{
	{"catalog_changes", "change_type"},
	{"config_changes", "change_type"},
	{"config_detail", "type"},
	{"config_names", "type"},
	{"config_summary", "type"},
	{"configs", "type"},

	// These are single column views
	{"analysis_types", ""},
	{"analyzer_types", ""},
	{"change_types", ""},
	{"config_classes", ""},
	{"config_types", ""},
}

func benchSizes() []int {
	raw := strings.TrimSpace(os.Getenv("DUTY_BENCH_SIZES"))
	if raw == "" {
		return defaultTestSizes
	}
	parts := strings.Split(raw, ",")
	sizes := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil || value <= 0 {
			continue
		}
		sizes = append(sizes, value)
	}
	if len(sizes) == 0 {
		return defaultTestSizes
	}
	return sizes
}

func BenchmarkRLS(b *testing.B) {
	for _, size := range benchSizes() {
		resetPG(b, false)
		_, err := setupConfigsForSize(testCtx, size)
		if err != nil {
			b.Fatalf("failed to setup configs for size %d: %v", size, err)
		}

		payloads := tagPayloads(b)
		b.Run(fmt.Sprintf("Sample-%d", size), func(b *testing.B) {
			for _, config := range benchConfigs {
				runBenchmark(b, config, payloads)
			}
			runRLSQueries(b, payloads[0], sampleTags[0])
		})
	}
}

// tagPayloads grants each sample tag set through the revision's native claim format.
func tagPayloads(b *testing.B) []pkgRLS.Payload {
	// the database outlives resetPG, so drop the Scopes an earlier size stored
	if err := testCtx.DB().Exec("DELETE FROM scopes WHERE namespace = 'bench'").Error; err != nil {
		b.Fatalf("failed to delete earlier bench scopes: %v", err)
	}

	g := gomega.NewWithT(b)
	var payloads []pkgRLS.Payload
	for _, tags := range sampleTags {
		var pairs []string
		for k, v := range tags {
			pairs = append(pairs, k+"="+v)
		}
		scope := models.Scope{ID: uuid.New(), Name: fmt.Sprintf("bench-tags-%d", len(payloads)), Namespace: "bench", Targets: types.JSON(`[]`)}
		if err := testCtx.DB().Create(&scope).Error; err != nil {
			b.Fatalf("failed to create scope for tags %v: %v", tags, err)
		}
		target := membership.Target{Type: policy.ResourceConfig, Selector: types.ResourceSelector{TagSelector: strings.Join(pairs, ",")}}
		if _, err := membership.Rebuild(testCtx, scope.ID, scope.Targets, []membership.Target{target}); err != nil {
			b.Fatalf("failed to store scope for tags %v: %v", tags, err)
		}

		// CI copies this benchmark to both revisions. Each Payload decoder retains only its native grant fields.
		claim, err := json.Marshal(map[string]any{
			"config": []map[string]any{{"scope": scope.ID.String(), "tags": tags}},
		})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		var payload pkgRLS.Payload
		g.Expect(json.Unmarshal(claim, &payload)).To(gomega.Succeed())
		payloads = append(payloads, payload)
	}
	return payloads
}

func runBenchmark(b *testing.B, config DistinctBenchConfig, payloads []pkgRLS.Payload) {
	b.Run(config.relation, func(b *testing.B) {
		for _, rls := range []bool{false, true} {
			resetPG(b, rls)
			name := "Without RLS"
			if rls {
				name = "With RLS"
			}

			// Testing out the performance when the RLS payload is also used as a WHERE clause
			// if rls && lo.Contains(viewsWithTags, config.relation) {
			// 	b.Run(name+"-With-Clause", func(b *testing.B) {
			// 		for i := 0; i < b.N; i++ {
			// 			b.StopTimer()
			// 			payload := pkgRLS.Payload{Tags: []map[string]string{sampleTags[i%len(sampleTags)]}}
			// 			if err := payload.SetPostgresSessionRLS(testCtx.DB(), false); err != nil {
			// 				b.Fatalf("failed to setup rls payload(%v): %v", payload, err)
			// 			}
			// 			b.StartTimer()

			// 			if result, err := fetchView(testCtx, config.relation, config.column, payload.Tags[0]); err != nil {
			// 				b.Fatalf("%v", err)
			// 			} else if result == 0 {
			// 				b.Fatalf("payload [%#v] got 0 results", payload)
			// 			}
			// 		}
			// 	})
			// 	resetPG(b, rls)
			// }

			b.Run(name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var payload pkgRLS.Payload
					if rls {
						b.StopTimer()
						payload = payloads[i%len(payloads)]
						if err := payload.SetGlobalPostgresSessionRLS(testCtx.DB()); err != nil {
							b.Fatalf("failed to setup rls payload with tag(%v): %v", payload, err)
						}

						if err := verifyRLSPayload(testCtx); err != nil {
							b.Fatalf("rls payload wasn't setup: %v", err)
						}
						b.StartTimer()
					}

					if result, err := fetchView(testCtx, config.relation, config.column, nil); err != nil {
						b.Fatalf("%v", err)
					} else if result == 0 {
						b.Fatalf("payload [%#v] got 0 results which doesn't seem right", payload)
					}
				}
			})
		}
	})
}

// runRLSQueries measures both full scans and selective reads: enumerating a Scope can help one and hurt the other.
func runRLSQueries(b *testing.B, payload pkgRLS.Payload, tags map[string]string) {
	b.Run("queries", func(b *testing.B) {
		resetPG(b, false)
		g := gomega.NewWithT(b)
		tagJSON, err := json.Marshal(tags)
		g.Expect(err).NotTo(gomega.HaveOccurred())

		// Derive expectations from the fixture's tags, independently of the membership predicate and RLS policy.
		var configs []struct {
			ID      uuid.UUID
			Allowed bool
		}
		g.Expect(testCtx.DB().Raw(`SELECT id, COALESCE(tags @> ?::jsonb, false) AS allowed FROM config_items`, string(tagJSON)).Scan(&configs).Error).To(gomega.Succeed())
		var allowedID, deniedID uuid.UUID
		var allowedCount int64
		for _, config := range configs {
			if config.Allowed {
				allowedID = config.ID
				allowedCount++
			} else {
				deniedID = config.ID
			}
		}
		g.Expect(allowedID).NotTo(gomega.Equal(uuid.Nil))
		g.Expect(deniedID).NotTo(gomega.Equal(uuid.Nil))

		for _, rls := range []bool{false, true} {
			resetPG(b, rls)
			name := "Without RLS"
			visibleCount, deniedCount := int64(len(configs)), int64(1)
			if rls {
				name = "With RLS"
				visibleCount, deniedCount = allowedCount, 0
				g.Expect(payload.SetGlobalPostgresSessionRLS(testCtx.DB())).To(gomega.Succeed())
				g.Expect(verifyRLSPayload(testCtx)).To(gomega.Succeed())
			}

			b.Run(name, func(b *testing.B) {
				for _, query := range []struct {
					name string
					sql  string
					args []any
					rows int64
				}{
					{"Count", "SELECT count(*) FROM config_items", nil, visibleCount},
					{"PointAllowed", "SELECT count(*) FROM config_items WHERE id = ?", []any{allowedID}, 1},
					{"PointDenied", "SELECT count(*) FROM config_items WHERE id = ?", []any{deniedID}, deniedCount},
					{"Page50", "SELECT count(*) FROM (SELECT id FROM config_items ORDER BY id LIMIT 50) AS page", nil, min(50, visibleCount)},
				} {
					b.Run(query.name, func(b *testing.B) {
						g := gomega.NewWithT(b)
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							var count int64
							g.Expect(testCtx.DB().Raw(query.sql, query.args...).Scan(&count).Error).To(gomega.Succeed())
							g.Expect(count).To(gomega.Equal(query.rows))
						}
					})
				}
			})
		}
	})
}
