package duty

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/utils"
	. "github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/postgrest"
	"github.com/spf13/pflag"
)

func BindPFlags(flags *pflag.FlagSet) {
	_ = flags.MarkDeprecated("postgrest-anon-role", "Use postgrest-role instead")
	flags.StringVar(&DefaultConfig.ConnectionString, "db", "DB_URL", "Connection string for the postgres database")
	flags.StringVar(&DefaultConfig.Schema, "db-schema", "public", "Postgres schema")
	flags.StringVar(&DefaultConfig.Postgrest.URL, "postgrest-uri", "http://localhost:3000", "URL for the PostgREST instance to use. If localhost is supplied, a PostgREST instance will be started")
	flags.StringVar(&DefaultConfig.Postgrest.LogLevel, "postgrest-log-level", "info", "PostgREST log level")
	flags.StringVar(&DefaultConfig.Postgrest.JWTSecret, "postgrest-jwt-secret", "PGRST_JWT_SECRET", "JWT Secret Token for PostgREST")
	flags.BoolVar(&DefaultConfig.SkipMigrations, "skip-migrations", false, "Skip database migrations")
	flags.BoolVar(&DefaultConfig.Postgrest.Disable, "disable-postgrest", false, "Disable PostgREST. Deprecated (Use --postgrest-uri '' to disable PostgREST)")
	flags.StringVar(&DefaultConfig.Postgrest.DBRole, "postgrest-role", "postgrest_api", "PostgREST role for authentication connections")
	flags.IntVar(&DefaultConfig.Postgrest.MaxRows, "postgrest-max-rows", 2000, "A hard limit to the number of rows PostgREST will fetch")
	flags.StringVar(&DefaultConfig.LogLevel, "db-log-level", "error", "Set gorm logging level. trace, debug & info")
	flags.BoolVar(&DefaultConfig.DisableKubernetes, "disable-kubernetes", false, "Disable Kubernetes integration")
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

	if config.Postgrest.URL != "" && !config.Postgrest.Disable {
		parsedURL, err := url.Parse(config.Postgrest.URL)
		if err != nil {
			logger.Fatalf("Failed to parse PostgREST URL: %v", err)
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

	stop := func() {}

	var ctx context.Context
	if c, err := InitDB(config); err != nil {
		return context.Context{}, stop, err
	} else {
		ctx = *c
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
