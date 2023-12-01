package duty

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/flanksource/commons/logger"
	dutyContext "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/migrate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var pool *pgxpool.Pool

var DefaultQueryTimeout = 30 * time.Second

func DefaultGormConfig() *gorm.Config {
	return &gorm.Config{
		FullSaveAssociations: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: NewGormLogger(LogLevel),
	}
}

// creates a new Gorm DB connection using the global pgx connection pool, must be called after NewPgxPool
func NewGorm(connection string, config *gorm.Config) (*gorm.DB, error) {
	db, err := NewDB(connection)
	if err != nil {
		return nil, err
	}

	return gorm.Open(
		gormpostgres.New(gormpostgres.Config{Conn: db}),
		config,
	)
}

func NewDB(connection string) (*sql.DB, error) {
	if strings.Contains(connection, "cloudsql-instance-connection-name") {
		parsed := parseParams(connection)
		userPrivateIP, _ := strconv.ParseBool(parsed["use-private-ip"])
		return cloudSQLConnect(context.TODO(), parsed["user"], parsed["db"], parsed["cloudsql-instance-connection-name"], userPrivateIP)
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

	if logger.IsTraceEnabled() {
		logrusLogger := &logrus.Logger{
			Out:          os.Stderr,
			Formatter:    new(logrus.TextFormatter),
			Hooks:        make(logrus.LevelHooks),
			Level:        logrus.DebugLevel,
			ExitFunc:     os.Exit,
			ReportCaller: false,
		}
		_ = logrusLogger
		//config.ConnConfig.Logger = logrusadapter.NewLogger(logrusLogger)
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

func cloudSQLConnect(ctx context.Context, user, dbName, instanceConnectionName string, usePrivate bool) (*sql.DB, error) {
	dialer, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, fmt.Errorf("cloudsqlconn.NewDialer: %w", err)
	}

	var opts []cloudsqlconn.DialOption
	if usePrivate {
		opts = append(opts, cloudsqlconn.WithPrivateIP())
	}

	dsn := fmt.Sprintf("user=%s database=%s", user, dbName)
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	config.DialFunc = func(ctx context.Context, network, instance string) (net.Conn, error) {
		return dialer.Dial(ctx, instanceConnectionName, opts...)
	}
	dbURI := stdlib.RegisterConnConfig(config)

	dbPool, err := sql.Open("pgx", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return dbPool, nil
}

// parseParams takes a string of key-value pairs separated by spaces and returns a map of parsed parameters.
func parseParams(input string) map[string]string {
	params := make(map[string]string)

	pairs := strings.Fields(input)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		}
	}

	return params
}
