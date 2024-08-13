package duty

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/api"
	dutyContext "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/drivers"
	dutyGorm "github.com/flanksource/duty/gorm"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/tracing"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var pool *pgxpool.Pool

var DefaultQueryTimeout = 30 * time.Second

// LogLevel is the log level for gorm logger
var LogLevel string

func Now() clause.Expr {
	return gorm.Expr("NOW()")
}

func Delete(ctx dutyContext.Context, model Table) error {
	return ctx.DB().Model(model).UpdateColumn("deleted_at", Now()).Error
}

type Table interface {
	TableName() string
}

func BindGoFlags() {
	flag.StringVar(&LogLevel, "db-log-level", "error", "Set gorm logging level. trace, debug & info")
}

func DefaultGormConfig() *gorm.Config {
	return &gorm.Config{
		FullSaveAssociations: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: dutyGorm.NewSqlLogger(logger.GetLogger("db")),
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

func getConnection(connection string) (string, error) {
	pgxConfig, err := drivers.ParseURL(connection)
	if err != nil {
		return connection, err
	} else if pgxConfig != nil {
		return stdlib.RegisterConnConfig(pgxConfig), nil
	}
	return connection, nil
}

func NewDB(connection string) (*sql.DB, error) {
	conn, err := getConnection(connection)
	if err != nil {
		return nil, err
	}
	return sql.Open("pgx", conn)
}

func NewPgxPool(connection string) (*pgxpool.Pool, error) {
	connection, err := getConnection(connection)
	if err != nil {
		return nil, err
	}

	pgUrl, err := url.Parse(connection)
	if err == nil {
		logger.Infof("Connecting to %s", pgUrl.Redacted())
	}

	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		return nil, err
	}

	config.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithIncludeQueryParameters(),
		// This option is required to enable the WithSpanNameFunc
		otelpgx.WithTrimSQLInSpanName(),
		otelpgx.WithSpanNameFunc(func(stmt string) string {
			// Trim span name after 80 chars
			maxL := 80
			if len(stmt) < maxL {
				maxL = len(stmt)
			}
			return stmt[:maxL]
		}),
	)

	// prevent deadlocks from concurrent queries
	if config.MaxConns < 20 {
		config.MaxConns = 20
	}

	pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	row := pool.QueryRow(context.Background(), "SELECT pg_size_pretty(pg_database_size($1));", config.ConnConfig.Database)
	var size string
	if err := row.Scan(&size); err != nil {
		return nil, err
	}

	logger.Infof("Initialized DB: %s (%s)", config.ConnConfig.Host, size)
	return pool, nil
}

func Migrate(config api.Config) error {
	db, err := NewDB(config.ConnectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := migrate.RunMigrations(db, config.ConnectionString, migrate.MigrateOptions{
		Skip:        config.SkipMigrations,
		IgnoreFiles: config.SkipMigrationFiles,
	}); err != nil {
		return err
	}

	// Reload postgrest schema after migrations
	if _, err := db.Exec(`NOTIFY pgrst, 'reload schema'`); err != nil {
		return err
	}

	return nil
}

// HasMigrationsRun performs a rudimentary check to see if the migrations have
// run at least once.
func HasMigrationsRun(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM migration_logs WHERE path = '099_optimize.sql'").Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func InitDB(config api.Config) (*dutyContext.Context, error) {
	db, pool, err := SetupDB(config)
	if err != nil {
		return nil, err
	}

	dutyctx := dutyContext.NewContext(context.Background()).WithDB(db, pool)

	statementTimeout := dutyctx.Properties().String("connection.statement_timeout", "60min")
	postgrestStatmentTimeout := dutyctx.Properties().String("connection.postgrest.statement_timeout", "60s")
	setStatementTimeouts(dutyctx, dutyctx.Pool(), config.ConnectionString, statementTimeout, postgrestStatmentTimeout)

	return &dutyctx, nil
}

// SetupDB runs migrations for the connection and returns a gorm.DB and a pgxpool.Pool
func SetupDB(config api.Config) (gormDB *gorm.DB, pgxpool *pgxpool.Pool, err error) {

	pgxpool, err = NewPgxPool(config.ConnectionString)
	if err != nil {
		return
	}

	conn, err := pgxpool.Acquire(context.TODO())
	if err != nil {
		return
	}
	defer conn.Release()

	if err := conn.Ping(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("error pinging database: %w", err)
	}

	gormDB, err = NewGorm(config.ConnectionString, DefaultGormConfig())
	if err != nil {
		return
	}

	if !config.SkipMigrations {
		if err = Migrate(config); err != nil {
			return
		}
	}

	return
}

func setStatementTimeouts(ctx context.Context, pool *pgxpool.Pool, connection string, connStatementTimeout, postgrestStatementTimeout string) {
	if _, err := pool.Exec(ctx, fmt.Sprintf(`ALTER ROLE postgrest_api SET statement_timeout = '%s'`, postgrestStatementTimeout)); err != nil {
		logger.Errorf("failed to set statement timeout for role postgrest_api: %v", err)
	}

	if _, err := pool.Exec(ctx, fmt.Sprintf(`ALTER ROLE postgrest_anon SET statement_timeout = '%s'`, postgrestStatementTimeout)); err != nil {
		logger.Errorf("failed to set statement timeout for role postgrest_anon: %v", err)
	}

	parsedConn, err := url.Parse(connection)
	if err != nil {
		return
	}

	user := parsedConn.User.Username()
	if user != "" && connStatementTimeout != "" {
		if _, err := pool.Exec(ctx, fmt.Sprintf(`ALTER ROLE %s SET statement_timeout = '%s'`, user, connStatementTimeout)); err != nil {
			logger.Errorf("failed to set statement timeout for role %q: %v", user, err)
		}
	}
}
