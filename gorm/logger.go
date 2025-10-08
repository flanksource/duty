package duty

import (
	"context"
	"errors"
	"fmt"
	"time"

	commons "github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// LogLevel log level
type LogLevel int

const (
	// Silent silent log level
	Silent LogLevel = iota + 1
	// Error error log level
	Error
	// Warn warn log level
	Warn
	// Info info log level
	Info
)

const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BlueBold    = "\033[34;1m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"
)

type Logger interface {
	LogMode(LogLevel) logger.Interface
	Info(context.Context, string, ...any)
	Warn(context.Context, string, ...any)
	Error(context.Context, string, ...any)
	Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}

type Config struct {
	SlowThreshold             time.Duration
	Colorful                  bool
	IgnoreRecordNotFoundError bool
	LogLevel                  int
}

type SqlLogger struct {
	Config
	commons.Logger
	traceParams bool
	maxLength   int
	baseLevel   commons.LogLevel
}

func (l *SqlLogger) WithLogLevel(level any) *SqlLogger {
	newlogger := *l
	newlogger.Logger = l.Logger.WithV(level)
	newlogger.baseLevel = commons.ParseLevel(newlogger, level)
	return &newlogger
}

func (l *SqlLogger) WithLogger(name string, level any) *SqlLogger {
	newlogger := *l
	newlogger.Logger = commons.GetLogger(name)
	newlogger.baseLevel = commons.ParseLevel(l.Logger, level)
	return &newlogger
}

func FromCommonsLevel(l commons.Logger, level any) logger.LogLevel {
	return logger.LogLevel(commons.ParseLevel(l, level))
}

func gormToCommonsLogLevel(level logger.LogLevel) commons.LogLevel {
	switch level {
	case logger.Info:
		return commons.Info
	case logger.Silent:
		return commons.Silent
	case logger.Warn:
		return commons.Warn
	case logger.Error:
		return commons.Error
	}
	return commons.Silent
}

func (l *SqlLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l.WithLogLevel(gormToCommonsLogLevel(level))
}

func NewSqlLogger(logger *commons.SlogLogger) logger.Interface {
	return &SqlLogger{
		Config: Config{
			Colorful:                  true,
			SlowThreshold:             properties.Duration(time.Second, "log.db.slowThreshold"),
			IgnoreRecordNotFoundError: true,
		},
		Logger:      logger,
		traceParams: logger.IsTraceEnabled() || properties.On(false, "log.db.params"),
		maxLength:   properties.Int(1024, "log.db.maxLength"),
		baseLevel:   commons.Silent,
	}
}

func (s SqlLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	s.Warnf(format, args...)
}

func (s SqlLogger) Info(ctx context.Context, format string, args ...interface{}) {
	s.Infof(format, args...)
}
func (s SqlLogger) Error(ctx context.Context, format string, args ...interface{}) {
	s.Errorf(format, args...)
}

var detailsFmt = Yellow + "[%dms] " + BlueBold + "[rows:%v]" + Reset + " %s"

// Trace print sql message
//
//nolint:cyclop
func (l *SqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if !l.IsLevelEnabled(commons.Error) {
		return
	}

	elapsed := time.Since(begin)
	msg := ""
	level := l.baseLevel

	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		sql = trunc(sql, l.maxLength)
		msg = fmt.Sprintf("ERROR >="+detailsFmt, elapsed/1e6, rows, err.Error()+" "+sql)
		level = commons.Error

	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		sql, rows := fc()
		sql = trunc(sql, l.maxLength)
		msg = fmt.Sprintf("SLOW SQL >= "+detailsFmt, elapsed/1e6, rows, sql)
		level = commons.Warn

	case l.LogLevel == int(commons.Info):
		sql, rows := fc()
		sql = trunc(sql, l.maxLength)

		msg = fmt.Sprintf(detailsFmt, elapsed/1e6, rows, sql)
	}
	if l.IsLevelEnabled(level) {
		l.V(level).Infof(msg)
	}
}

func trunc(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[0:length]
}

// ParamsFilter filter params
func (l *SqlLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.traceParams || l.GetLevel() >= commons.Info {
		return sql, params
	}
	return sql, nil
}
