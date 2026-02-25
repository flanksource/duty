package postgrest

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/flanksource/commons/exec"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/deps"
	"github.com/flanksource/duty/api"
)

func GoOffline() error {
	return runBinary(api.DefaultConfig, "--help")
}

func runBinary(config api.Config, msg string, args ...any) error {
	result, err := deps.Install("postgREST", config.Postgrest.Version, deps.WithBinDir(".bin"))
	if err != nil {
		return fmt.Errorf("failed to install postgREST: %w", err)
	}

	bin := filepath.Join(result.BinDir, "postgrest")

	env := map[string]string{
		"PGRST_SERVER_PORT":              strconv.Itoa(config.Postgrest.Port),
		"PGRST_DB_URI":                   config.ConnectionString,
		"PGRST_DB_SCHEMA":                config.Schema,
		"PGRST_DB_ANON_ROLE":             config.Postgrest.AnonDBRole,
		"PGRST_OPENAPI_SERVER_PROXY_URI": config.Postgrest.URL,
		"PGRST_LOG_LEVEL":                config.Postgrest.LogLevel,
		"PGRST_DB_MAX_ROWS":              strconv.Itoa(config.Postgrest.MaxRows),
		"PGRST_JWT_SECRET":               config.Postgrest.JWTSecret,
	}

	return exec.ExecfWithEnv(bin+" "+msg, env, args...)
}

func Start(config api.Config) {
	logger.Infof("Starting postgrest %s", config.Postgrest)
	if err := runBinary(config, ""); err != nil {
		logger.Errorf("Failed to start postgREST: %v", err)
	}
}

func PostgRESTEndpoint(config api.Config) string {
	return fmt.Sprintf("http://localhost:%d", config.Postgrest.Port)
}

func PostgRESTAdminEndpoint(config api.Config) string {
	return fmt.Sprintf("http://localhost:%d", config.Postgrest.AdminPort)
}
