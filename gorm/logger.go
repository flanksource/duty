package duty

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	commons "github.com/flanksource/commons/logger"
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
	Info(context.Context, string, ...interface{})
	Warn(context.Context, string, ...interface{})
	Error(context.Context, string, ...interface{})
	Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}

type Config struct {
	// SlowThreshold in nanoseconds
	SlowThreshold             int64
	Colorful                  bool
	IgnoreRecordNotFoundError bool
	LogLevel                  int
}

type SqlLogger struct {
	Config
	commons.Logger
	skipLevel int
}

func (l *SqlLogger) WithLogLevel(level any) *SqlLogger {
	newlogger := *l
	newlogger.Logger = l.Logger.WithV(level)
	return &newlogger
}

func FromCommonsLevel(l commons.Logger, level any) logger.LogLevel {
	return logger.LogLevel(commons.ParseLevel(l, level))
}

func (l *SqlLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l.WithLogLevel(level)
}

func NewSqlLogger(logger *commons.SlogLogger) logger.Interface {
	return &SqlLogger{
		Config: Config{
			Colorful:                  true,
			SlowThreshold:             time.Second.Nanoseconds(),
			IgnoreRecordNotFoundError: true,
		},
		Logger: logger,
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

	elapsed := int64(time.Since(begin).Nanoseconds())
	msg := ""
	level := commons.Info
	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		msg = fmt.Sprintf("ERROR >="+detailsFmt, elapsed/1e6, rows, err.Error()+" "+sql)
		level = commons.Error

	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		sql, rows := fc()
		msg = fmt.Sprintf("SLOW SQL >= "+detailsFmt, elapsed/1e6, rows, sql)
		level = commons.Warn

	case l.LogLevel == int(commons.Info):
		sql, rows := fc()

		switch strings.Trim(strings.Split(strings.ToLower(sql[0:int(math.Min(float64(len(sql)), 10))]), " ")[0], " \n") {
		case "select":
			if rows == 0 {
				level = commons.Trace1
			} else {
				level = commons.Trace
			}

		case "update", "insert", "delete":
			if rows == 0 {
				level = commons.Trace
			} else {
				level = commons.Debug

			}
		case "create", "alter", "drop":
			level = commons.Info
		default:
			level = commons.Debug
		}

		msg = fmt.Sprintf(detailsFmt, elapsed/1e6, rows, sql)
	}
	if l.IsLevelEnabled(level) {
		l.V(level).Infof(msg)
	}
}

// ParamsFilter filter params
func (l *SqlLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.GetLevel() >= commons.Debug {
		return sql, params
	}
	return sql, nil
}

type traceRecorder struct {
	Logger
	BeginAt      time.Time
	SQL          string
	RowsAffected int64
	Err          error
}

// New trace recorder
func (l *traceRecorder) New() *traceRecorder {
	return &traceRecorder{Logger: l.Logger, BeginAt: time.Now()}
}

// Trace implement logger interface
func (l *traceRecorder) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.BeginAt = begin
	l.SQL, l.RowsAffected = fc()
	l.Err = err
}
