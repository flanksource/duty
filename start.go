package duty

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/utils"
	. "github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/postgrest"
	"github.com/spf13/pflag"
	"gorm.io/plugin/prometheus"
)

func BindPFlags(flags *pflag.FlagSet) {
	_ = flags.MarkDeprecated("postgrest-anon-role", "Use postgrest-role instead")
	flags.StringVar(&DefaultConfig.ConnectionString, "db", "DB_URL", "Connection string for the postgres database")
	flags.StringVar(&DefaultConfig.Schema, "db-schema", "public", "Postgres schema")
	flags.StringVar(&DefaultConfig.Postgrest.URL, "postgrest-uri", "http://localhost:3000", "URL for the PostgREST instance to use. If localhost is supplied, a PostgREST instance will be started")
	flags.StringVar(&DefaultConfig.Postgrest.LogLevel, "postgrest-log-level", "info", "PostgREST log level")
	flags.StringVar(&DefaultConfig.Postgrest.JWTSecret, "postgrest-jwt-secret", "PGRST_JWT_SECRET", "JWT Secret Token for PostgREST")
	flags.BoolVar(&DefaultConfig.RunMigrations, "db-migrations", true, "Run database migrations")
	flags.BoolVar(&DefaultConfig.SkipMigrations, "skip-migrations", false, "Skip database migrations")
	flags.BoolVar(&DefaultConfig.Postgrest.Disable, "disable-postgrest", false, "Disable PostgREST. Deprecated (Use --postgrest-uri '' to disable PostgREST)")
	flags.StringVar(&DefaultConfig.Postgrest.DBRole, "postgrest-role", "postgrest_api", "PostgREST role for authentication connections")
	flags.IntVar(&DefaultConfig.Postgrest.MaxRows, "postgrest-max-rows", 2000, "A hard limit to the number of rows PostgREST will fetch")
	flags.StringVar(&DefaultConfig.LogLevel, "db-log-level", "error", "Set gorm logging level. trace, debug & info")
	flags.BoolVar(&DefaultConfig.DisableKubernetes, "disable-kubernetes", false, "Disable Kubernetes integration")
	flags.BoolVar(&DefaultConfig.Metrics, "db-metrics", false, "Expose db metrics")

	_ = flags.MarkDeprecated("db-migrations", "migrations are run by default. Use --skip-migrations to skip migrations.")
}

type StartOption func(config Config) Config

var DisablePostgrest = func(config Config) Config {
	config.Postgrest.Disable = true
	return config
}

var WithUrl = func(url string) func(config Config) Config {
	return func(config Config) Config {
		config.ConnectionString = url
		return config
	}
}

var SkipChangelogMigration = func(config Config) Config {
	config.SkipMigrationFiles = []string{"007_events.sql", "012_changelog_triggers_others.sql", "012_changelog_triggers_scrapers.sql"}
	return config
}

var EnableMetrics = func(config Config) Config {
	config.Metrics = true
	return config
}

var SkipMigrations = func(config Config) Config {
	config.SkipMigrations = true
	return config
}

var ClientOnly = func(config Config) Config {
	config.Postgrest.Disable = true
	config.SkipMigrations = true
	return config
}

var DisableKubernetes = func(config Config) Config {
	config.DisableKubernetes = true
	return config
}

func Start(name string, opts ...StartOption) (context.Context, func(), error) {
	config := DefaultConfig
	for _, opt := range opts {
		config = opt(config)
	}
	config = config.ReadEnv()

	stop := func() {}

	if strings.HasPrefix(config.ConnectionString, "embedded://") {
		embeddedDBConnectionString, stopper, err := embeddedDB("embedded", config.ConnectionString, uint32(FreePort()))
		if err != nil {
			return context.Context{}, nil, fmt.Errorf("failed to setup embedded postgres: %w", err)
		}

		stop = func() {
			if err := stopper(); err != nil {
				logger.Errorf("error stopping embedded postgres: %v", err)
			}
		}

		// override the embedded connection string with an actual postgres connection string
		config.ConnectionString = embeddedDBConnectionString
		DefaultConfig.ConnectionString = embeddedDBConnectionString
	}

	if config.Postgrest.URL != "" && !config.Postgrest.Disable {
		parsedURL, err := url.Parse(config.Postgrest.URL)
		if err != nil {
			return context.Context{}, nil, fmt.Errorf("failed to parse PostgREST URL: %v", err)
		}

		host := strings.ToLower(parsedURL.Hostname())
		port, _ := strconv.Atoi(parsedURL.Port())
		config.Postgrest.Port = int(port)
		if host == "localhost" {
			if config.Postgrest.JWTSecret == "" {
				logger.Warnf("PostgREST JWT secret not specified, generating random secret")
				config.Postgrest.JWTSecret = utils.RandomString(32)
			}
			go postgrest.Start(config)
		}
		DefaultConfig = config
	}

	var ctx context.Context
	if c, err := InitDB(config); err != nil {
		return context.Context{}, stop, err
	} else {
		ctx = *c
	}

	if config.Metrics {
		if err := ctx.DB().Use(prometheus.New(prometheus.Config{
			DBName:      ctx.Pool().Config().ConnConfig.Database,
			StartServer: false,
			MetricsCollector: []prometheus.MetricsCollector{
				&prometheus.Postgres{},
			},
		})); err != nil {
			return context.Context{}, stop, fmt.Errorf("failed to register prometheus metrics: %w", err)
		}
	}

	if !config.DisableKubernetes {
		if client, err := kubernetes.NewClient(); err == nil {
			ctx = ctx.WithKubernetes(client)
		} else {
			ctx.Infof("Kubernetes client not available: %v", err)
			ctx = ctx.WithKubernetes(kubernetes.Nil)
		}
	}

	return ctx, stop, nil
}

func embeddedDB(database, connectionString string, port uint32) (string, func() error, error) {
	embeddedPath := strings.TrimSuffix(strings.TrimPrefix(connectionString, "embedded://"), "/")
	if err := os.Chmod(embeddedPath, 0750); err != nil {
		logger.Errorf("failed to chmod %s: %v", embeddedPath, err)
	}

	logger.Infof("Starting embedded postgres server at %s", embeddedPath)

	embeddedPGServer := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(port).
		DataPath(path.Join(embeddedPath, "data")).
		RuntimePath(path.Join(embeddedPath, "runtime")).
		BinariesPath(path.Join(embeddedPath, "bin")).
		Version(embeddedpostgres.V14).
		Username("postgres").Password("postgres").
		Database(database))

	if err := embeddedPGServer.Start(); err != nil {
		return "", nil, fmt.Errorf("error starting embedded postgres: %w", err)
	}

	return fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, database), embeddedPGServer.Stop, nil
}

func FreePort() int {
	// Bind to port 0 to let the OS choose a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err.Error())
	}

	defer listener.Close()

	// Get the address of the listener
	address := listener.Addr().(*net.TCPAddr)
	return address.Port
}
