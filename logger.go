package duty

import (
	"context"
	"errors"
	"time"

	logsrusapi "github.com/sirupsen/logrus"

	"github.com/flanksource/commons/logger"
	"github.com/spf13/pflag"
	gLogger "gorm.io/gorm/logger"
)

// LogLevel is the log level for gorm logger
var LogLevel string

func BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&LogLevel, "db-log-level", "error", "Set gorm logging level. (error, warn, info)")
}

type gormLogger struct {
	logger                    logger.Logger
	LogLevel                  gLogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

func NewGormLogger() gLogger.Interface {
	l := logsrusapi.StandardLogger()
	l.SetFormatter(&logsrusapi.TextFormatter{
		ForceColors:  true,
		DisableQuote: true,
	})

	return &gormLogger{
		logger: logger.NewLogrusLogger(l),
	}
}

func (t *gormLogger) LogMode(level gLogger.LogLevel) gLogger.Interface {
	t.LogLevel = level

	switch level {
	case gLogger.Silent:
		t.logger.SetLogLevel(-1)
	case gLogger.Error:
		t.logger.SetLogLevel(1)
	case gLogger.Warn:
		t.logger.SetLogLevel(2)
	default:
		t.logger.SetLogLevel(3)
	}

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
	if l.LogLevel <= gLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gLogger.Error && (!errors.Is(err, gLogger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.logger.WithValues("rows", rows).Infof(sql)
		} else {
			l.logger.WithValues("rows", rows).Infof(sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gLogger.Warn:
		sql, rows := fc()
		if rows == -1 {
			l.logger.WithValues("Slow SQL", l.SlowThreshold).WithValues("rows", rows).Infof(sql)
		} else {
			l.logger.WithValues("Slow SQL", l.SlowThreshold).WithValues("rows", rows).Infof(sql)
		}
	case l.LogLevel == gLogger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.logger.WithValues("rows", rows).Infof(sql)
		} else {
			l.logger.WithValues("rows", rows).Infof(sql)
		}
	}
}
