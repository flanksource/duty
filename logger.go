package duty

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/flanksource/commons/logger"
	"github.com/spf13/pflag"
	gLogger "gorm.io/gorm/logger"
)

// LogLevel is the log level for gorm logger
var LogLevel string

func BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&LogLevel, "db-log-level", "error", "Set gorm logging level. error, warn & info")
}

type gormLogger struct {
	logger                    logger.Logger
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

func NewGormLogger() gLogger.Interface {
	l := logrus.StandardLogger()
	l.SetFormatter(&logrus.TextFormatter{
		ForceColors:  true,
		DisableQuote: true,
	})

	return &gormLogger{
		SlowThreshold: time.Second,
		logger:        logger.NewLogrusLogger(l),
	}
}

func (t *gormLogger) LogMode(level gLogger.LogLevel) gLogger.Interface {
	switch level {
	case gLogger.Silent:
		t.logger.SetLogLevel(int(logrus.FatalLevel))
	case gLogger.Error:
		t.logger.SetLogLevel(int(logrus.ErrorLevel))
	case gLogger.Warn:
		t.logger.SetLogLevel(int(logrus.WarnLevel))
	default:
		t.logger.SetLogLevel(int(logrus.InfoLevel))
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
