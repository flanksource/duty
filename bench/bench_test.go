package bench_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/flanksource/commons/logger"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/shutdown"
	"github.com/flanksource/duty/tests/setup"
)

var (
	testCtx context.Context
	connUrl string
)

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
