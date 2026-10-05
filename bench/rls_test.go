package bench_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

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

		scopes := tagScopes(b)
		b.Run(fmt.Sprintf("Sample-%d", size), func(b *testing.B) {
			for _, config := range benchConfigs {
				runBenchmark(b, config, scopes)
			}
		})
	}
}

// tagScopes stores one Scope per sample tag set, selecting the configs with those tags, and returns their ids.
func tagScopes(b *testing.B) []string {
	// the database outlives resetPG, so drop the Scopes an earlier size stored
	if err := testCtx.DB().Exec("DELETE FROM scopes WHERE namespace = 'bench'").Error; err != nil {
		b.Fatalf("failed to delete earlier bench scopes: %v", err)
	}

	var ids []string
	for _, tags := range sampleTags {
		var pairs []string
		for k, v := range tags {
			pairs = append(pairs, k+"="+v)
		}
		scope := models.Scope{ID: uuid.New(), Name: fmt.Sprintf("bench-tags-%d", len(ids)), Namespace: "bench", Targets: types.JSON(`[]`)}
		if err := testCtx.DB().Create(&scope).Error; err != nil {
			b.Fatalf("failed to create scope for tags %v: %v", tags, err)
		}
		target := membership.Target{Type: policy.ResourceConfig, Selector: types.ResourceSelector{TagSelector: strings.Join(pairs, ",")}}
		if _, err := membership.Rebuild(testCtx, scope.ID, scope.Targets, []membership.Target{target}); err != nil {
			b.Fatalf("failed to store scope for tags %v: %v", tags, err)
		}
		ids = append(ids, scope.ID.String())
	}
	return ids
}

func runBenchmark(b *testing.B, config DistinctBenchConfig, scopes []string) {
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
						grants := pkgRLS.NoRows()
						grants.Add(pkgRLS.Grant{Scope: scopes[i%len(scopes)]})
						payload = pkgRLS.Payload{Config: grants}
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
