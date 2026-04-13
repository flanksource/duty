package duty

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/flanksource/commons/logger"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/api"
	dutyContext "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/drivers"
	dutyGorm "github.com/flanksource/duty/gorm"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/tracing"
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

// NewGormFromPool creates a Gorm DB that reuses an existing *pgxpool.Pool.
// This is the preferred path for the main application: it shares the same
// connection pool (and therefore the ConnConfig.OnNotice handler, pgx
// Tracer, and MaxConns) with any direct pgxpool users on the same context.
//
// In particular, without sharing the pool, RAISE NOTICE / RAISE WARNING
// messages emitted by server-side functions are invisible to Go callers
// using ctx.DB() (GORM) because the stdlib driver creates its own pgx
// config with no notice handler.
//
// stdlib.OpenDBFromPool automatically sets db.SetMaxIdleConns(0), so GORM
// will not hoard connections from the pool.
func NewGormFromPool(pool *pgxpool.Pool, config *gorm.Config) (*gorm.DB, error) {
	db := stdlib.OpenDBFromPool(pool)

	gormDB, err := gorm.Open(
		gormpostgres.New(gormpostgres.Config{Conn: db}),
		config,
	)
	if err != nil {
		return nil, err
	}

	if err := gormDB.Use(tracing.NewPlugin()); err != nil {
		return nil, fmt.Errorf("error setting up tracing: %w", err)
	}

	return gormDB, nil
}

// SessionPropertyPrefix is the property-name prefix used to route values
// into Postgres session/transaction-local settings. A property
// "postgres.session.eu_debug.enabled=on" translates to
// `SET LOCAL eu_debug.enabled = 'on'` inside the provided transaction.
const SessionPropertyPrefix = "postgres.session."

// ApplySessionProperties runs `SET LOCAL <key> = '<value>'` inside the
// supplied GORM transaction for every property in `ctx.Properties()` whose
// key starts with SessionPropertyPrefix. The prefix is stripped before the
// SET is issued. Values are passed as text; Postgres will coerce them as
// needed by the GUC's type.
//
// The transaction MUST be an open tx (returned by `db.Begin()` or
// `db.WithContext(...).Begin()`); `SET LOCAL` outside a tx has no effect.
//
// Typical usage:
//
//	tx := ctx.DB().Begin()
//	if err := duty.ApplySessionProperties(ctx, tx); err != nil { ... }
//	defer tx.Rollback() // or commit
//	...
func ApplySessionProperties(ctx dutyContext.Context, tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("ApplySessionProperties: nil transaction")
	}
	settings := ctx.Properties().WithPrefix(SessionPropertyPrefix)
	if len(settings) == 0 {
		return nil
	}
	// Sort keys for deterministic apply order (helps tests and log output).
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := settings[k]
		// set_config(setting, new_value, is_local) is the parameterizable
		// equivalent of `SET LOCAL <key> = <value>` — it accepts the GUC
		// name as a text parameter, avoiding any need to escape/quote the
		// identifier ourselves.
		if err := tx.Exec("SELECT set_config(?, ?, true)", k, v).Error; err != nil {
			return fmt.Errorf("failed to set session property %q=%q: %w", k, v, err)
		}
	}
	return nil
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

	// Route Postgres NOTICE / WARNING messages (emitted via `RAISE NOTICE`
	// or `RAISE WARNING` from server-side functions) to the application
	// logger so server-side debug output is visible without needing to
	// attach psql. Severity → log level mapping mirrors Postgres semantics.
	config.ConnConfig.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		if n == nil {
			return
		}
		switch strings.ToUpper(n.Severity) {
		case "ERROR", "FATAL", "PANIC":
			logger.Errorf("pg %s: %s", n.Severity, n.Message)
		case "WARNING":
			logger.Warnf("pg %s: %s", n.Severity, n.Message)
		case "NOTICE", "INFO":
			logger.Infof("pg %s: %s", n.Severity, n.Message)
		case "LOG", "DEBUG":
			logger.Debugf("pg %s: %s", n.Severity, n.Message)
		default:
			logger.Infof("pg %s: %s", n.Severity, n.Message)
		}
	}

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

	if err := migrate.RunMigrations(db, config); err != nil {
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
func HasMigrationsRun(ctx dutyContext.Context) (bool, error) {
	var count int
	if err := ctx.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM migration_logs WHERE path = '099_optimize.sql'").Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func InitDB(config api.Config) (*dutyContext.Context, error) {
	db, pool, err := SetupDB(config)
	if err != nil {
		return nil, err
	}

	dutyctx := dutyContext.NewContext(context.Background()).WithDB(db, pool).WithConnectionString(config.ConnectionString)

	setStatementTimeouts(dutyctx, config)

	migrationsRan, err := HasMigrationsRun(dutyctx)
	if err != nil {
		return nil, fmt.Errorf("error querying migration logs: %w", err)
	}
	if !migrationsRan {
		return nil, fmt.Errorf("database migrations have not been run")
	}

	return &dutyctx, nil
}

// SetupDB runs migrations for the connection and returns a gorm.DB and a pgxpool.Pool
func SetupDB(config api.Config) (gormDB *gorm.DB, pgxpool *pgxpool.Pool, err error) {
	logger.Infof("Connecting to %s", config)

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

	cfg := DefaultGormConfig()

	if config.LogName != "" {
		cfg.Logger = dutyGorm.NewSqlLogger(logger.GetLogger(config.LogName))
	}

	// Share the pgxpool with GORM so Notice/Warning messages emitted by
	// server-side functions (RAISE NOTICE / RAISE WARNING) flow into the
	// pool's ConnConfig.OnNotice handler installed by NewPgxPool.
	gormDB, err = NewGormFromPool(pgxpool, cfg)
	if err != nil {
		return
	}

	if config.Migrate() {

		// Some triggers are dependent on kratos tables
		if config.KratosAuth {
			if err = verifyKratosMigration(gormDB); err != nil {
				return nil, nil, err
			}
		}

		if err = Migrate(config); err != nil {
			return
		}
	}

	return
}

func verifyKratosMigration(db *gorm.DB) error {
	var exists bool
	err := db.Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'identities')`).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("error confirming if kratos migration ran: %w", err)
	}
	if !exists {
		return fmt.Errorf("kratos created tables[identities] not found")
	}

	return nil
}

func setStatementTimeouts(ctx dutyContext.Context, config api.Config) {
	postgrestTimeout := ctx.Properties().Duration("db.postgrest.timeout", 1*time.Minute)

	if err := ctx.DB().Raw(fmt.Sprintf(`ALTER ROLE %s SET statement_timeout = '%0fs'`, config.Postgrest.DBRole, postgrestTimeout.Seconds())).Error; err != nil {
		logger.Errorf(err.Error())
	}

	if config.Postgrest.AnonDBRole != "" {
		if err := ctx.DB().Raw(fmt.Sprintf(`ALTER ROLE %s SET statement_timeout = '%0fs'`, config.Postgrest.AnonDBRole, postgrestTimeout.Seconds())).Error; err != nil {
			logger.Errorf(err.Error())
		}
	}

	statementTimeout := ctx.Properties().Duration("db.connection.timeout", 1*time.Hour)
	if username := config.GetUsername(); username != "" {
		if err := ctx.DB().Raw(fmt.Sprintf(`ALTER ROLE %s SET statement_timeout = '%0fs'`, username, statementTimeout.Seconds())).Error; err != nil {
			logger.Errorf(err.Error())
		}
	}
}
