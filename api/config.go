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
		Version:    "v10.0.0",
		DBAnonRole: "postgrest_api",
		Port:       3000,
		AdminPort:  3001,
		MaxRows:    2000,
	},
}

func NewConfig(connection string) Config {
	n := DefaultConfig
	n.ConnectionString = connection
	return n
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
	s := fmt.Sprintf("migrate=%v log=%v postgrest=(%s)", !c.SkipMigrations, c.LogLevel, c.Postgrest.String())
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
	JWTSecret  string
	DBRole     string
	AdminPort  int
	DBAnonRole string

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
