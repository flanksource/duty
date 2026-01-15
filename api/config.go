package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
)

var DefaultConfig = Config{
	Postgrest: PostgrestConfig{
		Version:      "v10.0.0",
		DBRole:       "postgrest_api",
		DBRoleBypass: "rls_bypasser",
		AnonDBRole:   "",
		Port:         3000,
		AdminPort:    3001,
		MaxRows:      2000,
	},
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

func PrintableSecret(secret string) string {
	if len(secret) == 0 {
		return "<nil>"
	} else if len(secret) > 30 {
		sum := md5.Sum([]byte(secret))
		hash := hex.EncodeToString(sum[:])
		return fmt.Sprintf("md5(%s),length=%d", hash[0:8], len(secret))
	} else if len(secret) > 16 {
		return fmt.Sprintf("%s****%s", secret[0:1], secret[len(secret)-2:])
	} else if len(secret) > 10 {
		return fmt.Sprintf("****%s", secret[len(secret)-1:])
	}
	return "****"
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
	Port      int
	Disable   bool
	LogLevel  string
	URL       string
	Version   string
	JWTSecret string
	AdminPort int

	// DBRole is the PostgREST role used for authenticated requests.
	DBRole string

	// DBRoleBypass is the PostgREST role used to bypass RLS for admin requests.
	DBRoleBypass string

	// AnonDBRole is the PostgREST role used for unauthenticated requests.
	AnonDBRole string

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
	return clone
}

func (p PostgrestConfig) String() string {
	return fmt.Sprintf("version:%v port=%d log-level=%v, jwt=%s",
		p.Version,
		p.Port,
		p.LogLevel,
		PrintableSecret(p.JWTSecret))
}
