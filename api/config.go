package api

var DefaultConfig = Config{
	Postgrest: PostgrestConfig{
		Version:    "v10.0.0",
		DBAnonRole: "postgrest_api",
		Port:       3000,
		MaxRows:    2000,
	},
}

type Config struct {
	ConnectionString, Schema string
	SkipMigrations           bool
	SkipMigrationFiles       []string
	DisableKubernetes        bool
	Namespace                string
	Postgrest                PostgrestConfig
	LogLevel                 string
	LogName                  string
}

type PostgrestConfig struct {
	Port       int
	Disable    bool
	LogLevel   string
	URL        string
	Version    string
	JWTSecret  string
	DBAnonRole string

	// A hard limit to the number of rows PostgREST will fetch from a view, table, or stored procedure.
	// Limits payload size for accidental or malicious requests.
	MaxRows int
}
