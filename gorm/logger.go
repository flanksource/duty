package duty

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/flanksource/commons/logger"
	gLogger "gorm.io/gorm/logger"
)

const Debug = "debug"
const Trace = "trace"

type gormLogger struct {
	logger                    logger.Logger
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

func NewGormLogger(level string) gLogger.Interface {
	if level == Trace {
		return gLogger.New(
			log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile), // io writer
			gLogger.Config{
				SlowThreshold:             time.Second,  // Slow SQL threshold
				LogLevel:                  gLogger.Info, // Log level
				IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      false,        // Don't include params in the SQL log
				Colorful:                  true,         // Disable color,
			},
		)
	}

	if level == Debug {
		return gLogger.New(
			log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile), // io writer
			gLogger.Config{
				SlowThreshold:             time.Second,  // Slow SQL threshold
				LogLevel:                  gLogger.Info, // Log level
				IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,         // Don't include params in the SQL log
				Colorful:                  true,         // Disable color
			},
		)
	}

	currentGormLogger := logger.StandardLogger().Named("db")

	switch level {
	case "trace":
		currentGormLogger.SetLogLevel(2)
	case "debug":
		currentGormLogger.SetLogLevel(1)
	default:
		currentGormLogger.SetLogLevel(0)
	}

	return &gormLogger{
		SlowThreshold: time.Second,
		logger:        currentGormLogger,
	}
}

// Pass the log level directly to NewGormLogger
func (t *gormLogger) LogMode(level gLogger.LogLevel) gLogger.Interface {
	// not applicable since the mapping of gorm's loglevel to common's logger's log level
	// doesn't work out well.
	return t
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Infof(msg, data)
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Warnf(msg, data)
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Errorf(msg, data)
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !l.logger.IsTraceEnabled() {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && (!errors.Is(err, gLogger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		l.logger.WithValues("elapsed", elapsed).WithValues("rows", rows).Errorf(sql)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		l.logger.WithValues("elapsed", elapsed).WithValues("slow SQL", l.SlowThreshold).WithValues("rows", rows).Warnf(sql)
	default:
		l.logger.WithValues("elapsed", elapsed).WithValues("rows", rows).Infof(sql)
	}
}
