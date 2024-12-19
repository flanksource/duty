package bench_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/google/uuid"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/shutdown"
	"github.com/flanksource/duty/tests/setup"
)

var (
	dbDataPath string
	genConfigs bool = true
	count           = 250_000

	tags      map[string]string
	testCtx   context.Context
	configIDs []uuid.UUID
)

func TestMain(m *testing.M) {
	shutdown.WaitForSignal()

	if v, ok := os.LookupEnv(setup.DUTY_DB_DATA_DIR); ok {
		dbDataPath = v
	}

	if v, ok := os.LookupEnv("DUTY_BENCH_DISABLE_CONFIG_GEN"); ok {
		genConfigs = v != ""
	}

	var err error

	args := []any{
		setup.WithoutDummyData, // we generate the dummy dataa
		setup.WithoutRLS,       // start without RLS
	}

	testCtx, err = setup.SetupDB("test", args...)
	if err != nil {
		shutdown.ShutdownAndExit(1, fmt.Sprintf("failed to setup db: %v", err))
	}

	{
		// This is required due to a bug in how we handle rls_enable / disable scripts.
		err := testCtx.DB().Exec("DELETE FROM migration_logs").Error
		if err != nil {
			shutdown.ShutdownAndExit(1, fmt.Sprintf("failed to delete migration logs: %v", err))
		}
	}

	if genConfigs {
		generatedList, err := generateConfigItems(testCtx, count)
		if err != nil {
			shutdown.ShutdownAndExit(1, err.Error())
		}

		configIDs = getAllConfigIDs(generatedList)
	}

	if dbDataPath != "" {
		if err := testCtx.DB().Select("id").Model(&models.ConfigItem{}).Find(&configIDs).Error; err != nil {
			shutdown.ShutdownAndExit(1, err.Error())
		}
		logger.Infof("fetched %d config ids from database", len(configIDs))
	}

	tags, err = setupRLSPayload(testCtx)
	if err != nil {
		shutdown.ShutdownAndExit(1, err.Error())
	}
	fmt.Println("using tags", tags)

	result := m.Run()

	shutdown.ShutdownAndExit(result, "exiting ...")
}

func BenchmarkFetchConfigNames(b *testing.B) {
	// generate a set of ids and use those same set of ids on both benchmarks
	sizes := []int{10_000, 25_000, 50_000, 100_000}
	var idBatches [][]uuid.UUID
	for _, s := range sizes {
		ids, err := shuffleAndPickNIDs(configIDs, s)
		if err != nil {
			b.Fatalf("failed to shuffle and pick ids: %v", err)
		}

		idBatches = append(idBatches, ids)
	}

	b.Run("WithoutRLS", func(b *testing.B) {
		benchFetchConfigNames(b, idBatches)
	})

	connUrl := testCtx.Value("db_url").(string)
	config := api.NewConfig(connUrl)
	if err := duty.Migrate(duty.EnableRLS(duty.RunMigrations(config))); err != nil {
		b.Fatalf("failed to enable rls: %v", err)
	}

	b.Run("WithRLS", func(b *testing.B) {
		benchFetchConfigNames(b, idBatches)
	})
}

func benchFetchConfigNames(b *testing.B, idBatches [][]uuid.UUID) {
	for _, ids := range idBatches {
		b.Run(fmt.Sprintf("FetchConfigNames-%d", len(ids)), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := fetchConfigNames(testCtx, ids); err != nil {
					b.Fatalf("%v", err)
				}
			}
		})

		if err := setup.RestartEmbeddedPG(); err != nil {
			b.Fatalf("failed to restart embedded pg")
		}

		if _, err := setupRLSPayload(testCtx); err != nil {
			b.Fatalf("failed to setup tags after restart: %v", err)
		}
	}
}
