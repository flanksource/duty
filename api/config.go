package api

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/flanksource/commons/logger"
)

var DefaultConfig = Config{
	Postgrest: PostgrestConfig{
		Version:    "v14.6",
		DBRole:     "postgrest_api",
		Arch:       runtime.GOARCH,
		AnonDBRole: "",
		Port:       3000,
		AdminPort:  3001,
		MaxRows:    2000,
	},
}

func init() {
	DefaultConfig.Postgrest = DefaultConfig.Postgrest.ReadEnv()
	v := DefaultConfig.Postgrest.Version

	if strings.HasPrefix(v, "v14") && v != "v14.1" && v != "v14.0" &&
		runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
		logger.Warnf("PostgREST v14.2+ does not have a darwin/arm64 binary, defaulting to v14.1 for darwin/amd64")
		DefaultConfig.Postgrest.Version = "v14.1"
	}
}

func NewConfig(connection string) Config {
	n := DefaultConfig
	n.ConnectionString = connection
	return n
}

type MigrationMode int

const (
	RunByDefault MigrationMode = iota
	SkipByDefault
)

type Config struct {
	Metrics                  bool
	ConnectionString, Schema string
	DisableKubernetes        bool
	Namespace                string
	Postgrest                PostgrestConfig
	LogLevel                 string
	LogName                  string

	EnableRLS          bool // Enable Row-level security
	DisableRLS         bool // Disable Row-level security
	RunMigrations      bool
	SkipMigrations     bool
	SkipMigrationFiles []string
	MigrationMode      MigrationMode

	// List of scripts that must run even if their hash hasn't changed.
	// Need just the filename without the `functions/` or `views/` prefix.
	MustRun []string

	// If we are using Kratos auth, some migrations
	// depend on kratos migrations being ran or not and
	// can cause problems if mission-control mirations run
	// before kratos migrations
	KratosAuth bool
}

func (t *Config) Migrate() bool {
	// We have two flags dictating whether to run migration or not
	// depending on whether it's being used on mission-control or (config-db/canary-checker).
	switch t.MigrationMode {
	case RunByDefault:
		return !t.SkipMigrations

	default:
		return t.RunMigrations
	}
}

func readEnv(val string) string {
	if v := os.Getenv(val); v != "" {
		return v
	}
	return val
}

func (c Config) ReadEnv() Config {
	clone := c
	clone.ConnectionString = readEnv(clone.ConnectionString)
	if clone.ConnectionString == "DB_URL" {
		clone.ConnectionString = ""
	}
	clone.Schema = readEnv(clone.Schema)
	clone.LogLevel = readEnv(clone.LogLevel)
	clone.Postgrest = clone.Postgrest.ReadEnv()
	return clone
}

func (c Config) String() string {
	s := fmt.Sprintf("migrate=%v log=%v postgrest=(%s)", c.Migrate(), c.LogLevel, c.Postgrest.String())
	if pgUrl, err := url.Parse(c.ConnectionString); err == nil {
		s = fmt.Sprintf("url=%s ", pgUrl.Redacted()) + s
	}

	return s
}

func (c Config) GetUsername() string {
	if url, err := url.Parse(c.ConnectionString); err != nil {
		return ""
	} else {
		return url.User.Username()
	}
}

type PostgrestConfig struct {
	Port       int
	Disable    bool
	LogLevel   string
	URL        string
	Version    string
	Arch       string
	JWTSecret  string
	DBRole     string
	AnonDBRole string
	AdminPort  int

	// A hard limit to the number of rows PostgREST will fetch from a view, table, or stored procedure.
	// Limits payload size for accidental or malicious requests.
	MaxRows int
}

func (p PostgrestConfig) ReadEnv() PostgrestConfig {
	clone := p

	clone.JWTSecret = readEnv(clone.JWTSecret)
	if clone.JWTSecret == "PGRST_JWT_SECRET" {
		clone.JWTSecret = ""
	}
	clone.LogLevel = readEnv(clone.LogLevel)

	if v := os.Getenv("PGRST_VERSION"); v != "" {
		clone.Version = v
	}
	if v := os.Getenv("PGRST_ARCH"); v != "" {
		clone.Arch = v
	}
	return clone
}

func (p PostgrestConfig) String() string {
	return fmt.Sprintf("version:%v port=%d log-level=%v, jwt=%s",
		p.Version,
		p.Port,
		p.LogLevel,
		logger.PrintableSecret(p.JWTSecret))
}
