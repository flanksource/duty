package bench_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/shutdown"
	"github.com/flanksource/duty/tests/setup"
)

var (
	dbDataPath string
	disableRLS bool
	count      = 250_000

	testCtx   context.Context
	configIDs []uuid.UUID
)

func TestMain(m *testing.M) {
	shutdown.WaitForSignal()

	if v, ok := os.LookupEnv(setup.DUTY_DB_DATA_DIR); ok {
		dbDataPath = v
	}

	var err error

	args := []any{setup.WithoutDummyData} // we generate the dummy dataa
	if disableRLS {
		args = append(args, setup.WithoutRLS)
	}

	// Setup DB without RLS first
	testCtx, err = setup.SetupDB("test", args...)
	if err != nil {
		shutdown.ShutdownAndExit(1, fmt.Sprintf("failed to setup db: %v", err))
	}

	if dbDataPath == "" {
		generatedList, err := generateConfigItems(testCtx, count)
		if err != nil {
			shutdown.ShutdownAndExit(1, err.Error())
		}

		configIDs = getAllConfigIDs(generatedList)
	} else {
		if err := testCtx.DB().Select("id").Model(&models.ConfigItem{}).Find(&configIDs).Error; err != nil {
			shutdown.ShutdownAndExit(1, err.Error())
		}
		logger.Infof("fetched %d config ids from database", len(configIDs))
	}

	if err := setupRLSPayload(testCtx); err != nil {
		shutdown.ShutdownAndExit(1, err.Error())
	}

	result := m.Run()

	shutdown.ShutdownAndExit(result, "exiting ...")
}

func BenchmarkFetchConfigNames(b *testing.B) {
	sizes := []int{10_000, 25_000, 50_000, 100_000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("FetchConfigNames-%d", size), func(b *testing.B) {
			ids := shuffleAndPickNIDs(configIDs, size)

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
	}
}

// func BenchmarkFetchConfigTypes(b *testing.B) {
// 	b.Run("FetchConfigTypes", func(b *testing.B) {
// 		b.ResetTimer()
// 		for i := 0; i < b.N; i++ {
// 			if err := fetchConfigTypes(testCtx); err != nil {
// 				b.Fatalf("%v", err)
// 			}
// 		}
// 	})
// }
