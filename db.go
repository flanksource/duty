package duty

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var pool *pgxpool.Pool

var DefaultQueryTimeout = 30 * time.Second

func DefaultGormConfig() *gorm.Config {
	logConfig := glogger.Config{
		SlowThreshold:             time.Second,   // Slow SQL threshold
		LogLevel:                  glogger.Error, // Log level
		IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
	}

	if logger.IsDebugEnabled() {
		logConfig.LogLevel = glogger.Warn
	}
	if logger.IsTraceEnabled() {
		logConfig.LogLevel = glogger.Info
	}

	return &gorm.Config{
		FullSaveAssociations: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: glogger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags), // io writer
			logConfig),
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
	return sql.Open("pgx", connection)
}

func NewPgxPool(connection string) (*pgxpool.Pool, error) {
	if pool != nil {
		return pool, nil
	}

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
	return migrate.RunMigrations(db, connection, *migrateOptions)
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
