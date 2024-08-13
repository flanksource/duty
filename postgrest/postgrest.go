package postgrest

import (
	"strconv"

	"github.com/flanksource/commons/deps"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/api"
)

func GoOffline() error {
	return getBinary(api.DefaultConfig)("--help")
}

func getBinary(config api.Config) deps.BinaryFunc {
	return deps.BinaryWithEnv("postgREST", config.Postgrest.Version, ".bin", map[string]string{
		"PGRST_SERVER_PORT":              strconv.Itoa(config.Postgrest.Port),
		"PGRST_DB_URI":                   config.ConnectionString,
		"PGRST_DB_SCHEMA":                config.Schema,
		"PGRST_DB_ANON_ROLE":             config.Postgrest.DBAnonRole,
		"PGRST_OPENAPI_SERVER_PROXY_URI": config.Postgrest.URL,
		"PGRST_LOG_LEVEL":                config.Postgrest.LogLevel,
		"PGRST_DB_MAX_ROWS":              strconv.Itoa(config.Postgrest.MaxRows),
		"PGRST_JWT_SECRET":               config.Postgrest.JWTSecret,
	})
}

func Start(config api.Config) {
	logger.Infof("Starting postgrest server on port %s", config.Postgrest.Port)
	if err := getBinary(config)(""); err != nil {
		logger.Errorf("Failed to start postgREST: %v", err)
	}
}
