package bench_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	pkgRLS "github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/shutdown"
	"github.com/flanksource/duty/tests/setup"
)

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

var (
	testCtx context.Context
	connUrl string

	// number of total configs in the database
	defaultTestSizes = []int{10_000, 25_000, 50_000, 100_000}
)

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

func setupTestDB(dbPath string) error {
	logger.Infof("using %q as the pg data dir", dbPath)
	os.Setenv(setup.DUTY_DB_DATA_DIR, dbPath)

	shutdown.AddHookWithPriority("delete data dir", shutdown.PriorityCritical+1, func() {
		if err := os.RemoveAll(dbPath); err != nil {
			logger.Errorf("failed to delete data dir: %v", err)
		}
	})

	var err error
	testCtx, err = setup.SetupDB("test",
		setup.WithoutDummyData, // we generate the dummy data
		setup.WithoutRLS,       // start without RLS
	)
	if err != nil {
		return fmt.Errorf("failed to setup db: %v", err)
	}
	connUrl = testCtx.Value("db_url").(string)
	return nil
}

func TestMain(m *testing.M) {
	shutdown.WaitForSignal()

	// Create a fixed postgres data directory
	dbDataPath, err := os.CreateTemp("", "bench-pg-dir-*")
	if err != nil {
		shutdown.ShutdownAndExit(1, "failed to create temp dir for db")
	}

	if err := setupTestDB(dbDataPath.Name()); err != nil {
		shutdown.ShutdownAndExit(1, err.Error())
	}

	result := m.Run()
	shutdown.ShutdownAndExit(result, "exiting ...")
}

func BenchmarkMain(b *testing.B) {
	for _, size := range benchSizes() {
		resetPG(b, false)
		_, err := setupConfigsForSize(testCtx, size)
		if err != nil {
			b.Fatalf("failed to setup configs for size %d: %v", size, err)
		}

		b.Run(fmt.Sprintf("Sample-%d", size), func(b *testing.B) {
			for _, config := range benchConfigs {
				runBenchmark(b, config)
			}
		})
	}
}

func runBenchmark(b *testing.B, config DistinctBenchConfig) {
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
						payload = pkgRLS.Payload{Scopes: []uuid.UUID{benchScopeIDs[i%len(benchScopeIDs)]}}
						if err := payload.SetGlobalPostgresSessionRLS(testCtx.DB()); err != nil {
							b.Fatalf("failed to setup rls payload with scope(%v): %v", payload, err)
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
