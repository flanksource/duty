package postgrest

import (
	"fmt"
	"strconv"

	"github.com/flanksource/commons/deps"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/api"
)

func GoOffline() error {
	return getBinary(api.DefaultConfig)("--help")
}

func getBinary(config api.Config) deps.BinaryFunc {
	opts := map[string]string{
		"PGRST_SERVER_PORT":              strconv.Itoa(config.Postgrest.Port),
		"PGRST_DB_URI":                   config.ConnectionString,
		"PGRST_DB_SCHEMA":                config.Schema,
		"PGRST_DB_ANON_ROLE":             config.Postgrest.AnonDBRole,
		"PGRST_OPENAPI_SERVER_PROXY_URI": config.Postgrest.URL,
		"PGRST_LOG_LEVEL":                config.Postgrest.LogLevel,
		"PGRST_DB_MAX_ROWS":              strconv.Itoa(config.Postgrest.MaxRows),
		"PGRST_JWT_SECRET":               config.Postgrest.JWTSecret,
	}
	return deps.BinaryWithEnv("postgREST", config.Postgrest.Version, ".bin", opts)
}

func Start(config api.Config) {
	logger.Infof("Starting postgrest %s", config.Postgrest)
	if err := getBinary(config)(""); err != nil {
		logger.Errorf("Failed to start postgREST: %v", err)
	}
}

func PostgRESTEndpoint(config api.Config) string {
	return fmt.Sprintf("http://localhost:%d", config.Postgrest.Port)
}

func PostgRESTAdminEndpoint(config api.Config) string {
	return fmt.Sprintf("http://localhost:%d", config.Postgrest.AdminPort)
}
