package duty

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	extraClausePlugin "github.com/WinterYukky/gorm-extra-clause-plugin"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
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

var DisableRLS = func(config api.Config) api.Config {
	config.DisableRLS = true
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

		stop = stopper

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
			dbStop := stop
			stop = func() {
				c.Pool().Close()
				dbStop()
			}
		}
	}

	if ctx.DB() != nil {
		_ = ctx.DB().Use(extraClausePlugin.New())
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
		ctx = ctx.WithKubernetes(connection.KubernetesConnection{})
	}

	return ctx, stop, nil
}

const posmasterLinePort = 3

func embeddedDB(database, connectionString string, port uint32) (string, func(), error) {
	embeddedPath := strings.TrimSuffix(strings.TrimPrefix(connectionString, "embedded://"), "/")
	if err := os.Chmod(embeddedPath, 0750); err != nil {
		logger.Errorf("failed to chmod %s: %v", embeddedPath, err)
	}

	dataPath := path.Join(embeddedPath, "data")
	pgVersion := embeddedpostgres.V16
	if _, err := os.Stat(dataPath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(dataPath, 0750); err != nil {
			logger.Errorf("failed to create data dir %s: %v", dataPath, err)
		}
	} else {
		pgVersionBytes, err := os.ReadFile(path.Join(dataPath, "PG_VERSION"))
		if err != nil {
			logger.Errorf("error reading PG_VERSION file: %v", err)
		} else {
			switch strings.TrimSpace(string(pgVersionBytes)) {
			case "14":
				pgVersion = embeddedpostgres.V14
			case "15":
				pgVersion = embeddedpostgres.V15
			case "16":
				pgVersion = embeddedpostgres.V16
			}
		}
	}

	logger.Infof("Starting embedded postgres server at %s", embeddedPath)

	embeddedPGServer := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(port).
		DataPath(dataPath).
		RuntimePath(path.Join(embeddedPath, "runtime")).
		BinariesPath(path.Join(embeddedPath, "bin")).
		Version(pgVersion).
		Username("postgres").Password("postgres").
		Database(database))

	stop := func() {
		logger.Infof("Stopping embedded db")
		if err := embeddedPGServer.Stop(); err != nil {
			logger.Errorf(err.Error())
		}
	}

	if err := embeddedPGServer.Start(); err != nil {
		if strings.Contains(err.Error(), "Is another postmaster") && strings.Contains(err.Error(), "running in data directory") {
			postMasterOpts := path.Join(embeddedPath, "data", "postmaster.pid")
			if opts := files.SafeRead(postMasterOpts); opts != "" {
				args := strings.Split(opts, "\n")
				portString := args[posmasterLinePort]
				if p, err := strconv.ParseUint(portString, 10, 32); err == nil {
					port = uint32(p)
					logger.Infof("Postgres already running on %d", port)
				}
			}
		} else {
			return "", nil, fmt.Errorf("error starting embedded postgres: %w", err)
		}
	}

	return fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, database), stop, nil
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
