package duty

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/flanksource/commons/logger"
	dutyContext "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/drivers"
	dutyGorm "github.com/flanksource/duty/gorm"
	"github.com/flanksource/duty/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/pflag"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

var pool *pgxpool.Pool

var DefaultQueryTimeout = 30 * time.Second

// LogLevel is the log level for gorm logger
var LogLevel string

func BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&LogLevel, "db-log-level", "error", "Set gorm logging level. trace, debug & info")
}

func DefaultGormConfig() *gorm.Config {
	return &gorm.Config{
		FullSaveAssociations: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: dutyGorm.NewGormLogger(LogLevel),
	}
}

// creates a new Gorm DB connection using the global pgx connection pool, must be called after NewPgxPool
func NewGorm(connection string, config *gorm.Config) (*gorm.DB, error) {
	db, err := NewDB(connection)
	if err != nil {
		return nil, err
	}

	Gorm, err := gorm.Open(
		gormpostgres.New(gormpostgres.Config{Conn: db}),
		config,
	)
	if err != nil {
		return nil, err
	}

	if err := Gorm.Use(tracing.NewPlugin()); err != nil {
		return nil, fmt.Errorf("error setting up tracing: %w", err)
	}

	return Gorm, nil
}

func NewDB(connection string) (*sql.DB, error) {
	pgxConfig, err := drivers.ParseURL(connection)
	if err != nil {
		return nil, err
	} else if pgxConfig != nil {
		connection = stdlib.RegisterConnConfig(pgxConfig)
	}

	return sql.Open("pgx", connection)
}

func NewPgxPool(connection string) (*pgxpool.Pool, error) {
	pgUrl, err := url.Parse(connection)
	if err != nil {
		return nil, err
	}

	logger.Infof("Connecting to %s", pgUrl.Redacted())

	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		return nil, err
	}

	// prevent deadlocks from concurrent queries
	if config.MaxConns < 20 {
		config.MaxConns = 20
	}

	pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	row := pool.QueryRow(context.TODO(), "SELECT pg_size_pretty(pg_database_size($1));", config.ConnConfig.Database)
	var size string
	if err := row.Scan(&size); err != nil {
		return nil, err
	}

	logger.Infof("Initialized DB: %s (%s)", config.ConnConfig.Host, size)
	return pool, nil
}

func Migrate(connection string, opts *migrate.MigrateOptions) error {
	db, err := NewDB(connection)
	if err != nil {
		return err
	}
	defer db.Close()

	migrateOptions := opts
	if migrateOptions == nil {
		migrateOptions = &migrate.MigrateOptions{}
	}
	if err := migrate.RunMigrations(db, connection, *migrateOptions); err != nil {
		return err
	}

	// Reload postgrest schema after migrations
	if _, err := db.Exec(`NOTIFY pgrst, 'reload schema'`); err != nil {
		return err
	}

	return nil
}

func InitDB(connection string, migrateOpts *migrate.MigrateOptions) (*dutyContext.Context, error) {
	db, pool, err := SetupDB(connection, migrateOpts)
	if err != nil {
		return nil, err
	}
	ctx := dutyContext.NewContext(context.Background()).WithDB(db, pool)
	return &ctx, nil
}

// SetupDB runs migrations for the connection and returns a gorm.DB and a pgxpool.Pool
func SetupDB(connection string, migrateOpts *migrate.MigrateOptions) (gormDB *gorm.DB, pgxpool *pgxpool.Pool, err error) {
	pgxpool, err = NewPgxPool(connection)
	if err != nil {
		return
	}

	conn, err := pgxpool.Acquire(context.Background())
	if err != nil {
		return
	}
	defer conn.Release()

	gormDB, err = NewGorm(connection, DefaultGormConfig())
	if err != nil {
		return
	}

	if err = Migrate(connection, migrateOpts); err != nil {
		return
	}

	return
}
