package duty

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var connectionString string
var pool *pgxpool.Pool

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
		Logger: glogger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags), // io writer
			logConfig),
	}
}

// creates a new Gorm DB connection using the global pgx connection pool, must be called after NewPgxPool
func NewGorm(config *gorm.Config) (*gorm.DB, error) {
	db, err := NewDB()
	if err != nil {
		return nil, err
	}

	return gorm.Open(
		gormpostgres.New(gormpostgres.Config{Conn: db}),
		config,
	)
}

func NewDB() (*sql.DB, error) {
	return sql.Open("pgx", connectionString)
}

func NewPgxPool(connection string) (*pgxpool.Pool, error) {
	connectionString = connection
	logger.Errorf("Connecting to %s", connection)
	if pool != nil {
		return pool, nil
	}
	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		return nil, err
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
		config.ConnConfig.Logger = logrusadapter.NewLogger(logrusLogger)
	}
	pool, err = pgxpool.ConnectConfig(context.Background(), config)
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
