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
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/postgrest"
	"github.com/spf13/pflag"
	"gorm.io/plugin/prometheus"
)

func BindPFlags(flags *pflag.FlagSet, opts ...StartOption) {
	config := api.DefaultConfig
	for _, opt := range opts {
		config = opt(config)
	}

	_ = flags.MarkDeprecated("postgrest-anon-role", "Use postgrest-role instead")
	flags.StringVar(&api.DefaultConfig.ConnectionString, "db", "DB_URL", "Connection string for the postgres database")
	flags.StringVar(&api.DefaultConfig.Schema, "db-schema", "public", "Postgres schema")
	flags.StringVar(&api.DefaultConfig.Postgrest.URL, "postgrest-uri", "http://localhost:3000", "URL for the PostgREST instance to use. If localhost is supplied, a PostgREST instance will be started")
	flags.StringVar(&api.DefaultConfig.Postgrest.LogLevel, "postgrest-log-level", "info", "PostgREST log level")
	flags.StringVar(&api.DefaultConfig.Postgrest.JWTSecret, "postgrest-jwt-secret", "PGRST_JWT_SECRET", "JWT Secret Token for PostgREST")
	flags.BoolVar(&api.DefaultConfig.Postgrest.Disable, "disable-postgrest", config.Postgrest.Disable, "Disable PostgREST. Deprecated (Use --postgrest-uri '' to disable PostgREST)")
	flags.StringVar(&api.DefaultConfig.Postgrest.DBRole, "postgrest-role", "postgrest_api", "PostgREST role for authentication connections")
	flags.StringVar(&api.DefaultConfig.Postgrest.AnonDBRole, "postgrest-anon-role", "postgrest_anon", "PostgREST role for unauthenticated connections")

	flags.IntVar(&api.DefaultConfig.Postgrest.MaxRows, "postgrest-max-rows", 2000, "A hard limit to the number of rows PostgREST will fetch")
	flags.StringVar(&api.DefaultConfig.LogLevel, "db-log-level", "error", "Set gorm logging level. trace, debug & info")
	flags.BoolVar(&api.DefaultConfig.DisableKubernetes, "disable-kubernetes", false, "Disable Kubernetes integration")
	flags.BoolVar(&api.DefaultConfig.Metrics, "db-metrics", false, "Expose db metrics")

	if config.MigrationMode == api.SkipByDefault {
		flags.BoolVar(&api.DefaultConfig.RunMigrations, "db-migrations", false, "Run database migrations")
	} else {
		flags.BoolVar(&api.DefaultConfig.SkipMigrations, "skip-migrations", false, "Skip database migrations")
		flags.BoolVar(&api.DefaultConfig.RunMigrations, "db-migrations", true, "Run database migrations")
		_ = flags.MarkDeprecated("db-migrations", "migrations are run by default. Use --skip-migrations to skip migrations.")
	}
}

type StartOption func(config api.Config) api.Config

var EnableRLS = func(config api.Config) api.Config {
	config.EnableRLS = true
	return config
}

var DisablePostgrest = func(config api.Config) api.Config {
	config.Postgrest.Disable = true
	return config
}

var WithUrl = func(url string) func(config api.Config) api.Config {
	return func(config api.Config) api.Config {
		config.ConnectionString = url
		return config
	}
}

var SkipMigrationByDefaultMode = func(config api.Config) api.Config {
	config.MigrationMode = api.SkipByDefault
	return config
}

var SkipChangelogMigration = func(config api.Config) api.Config {
	config.SkipMigrationFiles = []string{"007_events.sql", "012_changelog_triggers_others.sql", "012_changelog_triggers_scrapers.sql"}
	return config
}

var EnableMetrics = func(config api.Config) api.Config {
	config.Metrics = true
	return config
}

var RunMigrations = func(config api.Config) api.Config {
	config.MigrationMode = api.SkipByDefault
	config.RunMigrations = true
	return config
}

var SkipMigrations = func(config api.Config) api.Config {
	config.MigrationMode = api.RunByDefault
	config.RunMigrations = false
	return config
}

var ClientOnly = func(config api.Config) api.Config {
	config.Postgrest.Disable = true
	config.SkipMigrations = true
	return config
}

var DisableKubernetes = func(config api.Config) api.Config {
	config.DisableKubernetes = true
	return config
}

var KratosAuth = func(config api.Config) api.Config {
	config.KratosAuth = true
	return config
}

func Start(name string, opts ...StartOption) (context.Context, func(), error) {
	config := api.DefaultConfig
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
		api.DefaultConfig.ConnectionString = embeddedDBConnectionString
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
		api.DefaultConfig = config
	}

	var ctx context.Context
	if config.ConnectionString == "" {
		logger.Warnf("--db not configured")
		ctx = context.New()
	} else {
		if c, err := InitDB(config); err != nil {
			return context.Context{}, stop, err
		} else {
			ctx = *c
			stop = func() {
				c.Pool().Close()
			}
		}
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
		if client, config, err := kubernetes.NewClient(logger.GetLogger("k8s")); err == nil {
			ctx = ctx.WithKubernetes(client, config)
		} else {
			ctx.Infof("Kubernetes client not available: %v", err)
			ctx = ctx.WithKubernetes(kubernetes.Nil, nil)
		}
	}

	return ctx, stop, nil
}

func embeddedDB(database, connectionString string, port uint32) (string, func() error, error) {
	embeddedPath := strings.TrimSuffix(strings.TrimPrefix(connectionString, "embedded://"), "/")
	if err := os.Chmod(embeddedPath, 0750); err != nil {
		logger.Errorf("failed to chmod %s: %v", embeddedPath, err)
	}

	dataPath := path.Join(embeddedPath, "data")
	if err := os.MkdirAll(dataPath, 750); err != nil {
		logger.Errorf("failed to create data dir %s: %v", dataPath, err)
	}

	logger.Infof("Starting embedded postgres server at %s", embeddedPath)

	embeddedPGServer := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(port).
		DataPath(dataPath).
		RuntimePath(path.Join(embeddedPath, "runtime")).
		BinariesPath(path.Join(embeddedPath, "bin")).
		Version(embeddedpostgres.V15).
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
