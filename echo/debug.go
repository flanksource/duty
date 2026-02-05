package echo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/commons/timer"
	"github.com/google/gops/agent"
	"github.com/labstack/echo/v4"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/robfig/cron/v3"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/shutdown"
)

var Crons = cmap.New[*cron.Cron]()

func RegisterCron(cron *cron.Cron) {

	// Cache cron objects by their pointer
	Crons.SetIfAbsent(fmt.Sprintf("%p", cron), cron)
}

func init() {
	// disables default handlers registered by importing net/http/pprof.
	http.DefaultServeMux = http.NewServeMux()

	if err := agent.Listen(agent.Options{}); err != nil {
		logger.Errorf(err.Error())
	}

	// stop scheduling
	shutdown.AddHookWithPriority("cron scheduler", shutdown.PriorityJobs-1, func() {
		for cronScheduler := range Crons.IterBuffered() {
			cronScheduler.Val.Stop()
		}
	})
}

// RestrictToLocalhost is a middleware that restricts access to localhost
func RestrictToLocalhost(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		remoteIP := net.ParseIP(c.RealIP())
		if remoteIP == nil {
			return echo.NewHTTPError(http.StatusForbidden, "Invalid IP address")
		}

		if !remoteIP.IsLoopback() {
			return echo.NewHTTPError(http.StatusForbidden, "Access restricted to localhost")
		}

		return next(c)
	}
}

func AddDebugHandlers(ctx context.Context, e *echo.Echo, rbac echo.MiddlewareFunc) {

	// Add pprof routes with localhost restriction
	pprofGroup := e.Group("/debug/pprof")
	pprofGroup.Use(RestrictToLocalhost)
	pprofGroup.GET("/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	pprofGroup.GET("/cmdline*", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	pprofGroup.GET("/profile*", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	pprofGroup.GET("/symbol*", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	pprofGroup.GET("/trace*", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	debug := e.Group("/debug", rbac)

	debug.GET("/routes", func(c echo.Context) error {
		return c.JSON(http.StatusOK, e.Routes())
	})

	debug.GET("/loggers", func(c echo.Context) error {
		return c.JSON(200, logger.GetNamedLoggingLevels())
	})

	debug.POST("/loggers", func(c echo.Context) error {
		logName := c.Request().FormValue("logger")
		logLevel := c.Request().FormValue("level")
		duration := c.Request().FormValue("duration")
		currentLevel := logger.GetLogger(logName).GetLevel()
		if duration != "" {
			durationInt, err := time.ParseDuration(duration)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err)
			}
			logger.Infof("Setting logger %s level to %s for %v", logName, logLevel, duration)

			go func() {
				time.Sleep(durationInt)
				logger.GetLogger(logName).SetLogLevel(currentLevel)
			}()
		}
		if logName != "" && logLevel != "" {
			if duration == "" {
				logger.Infof("Setting logger %s level to %s", logName, logLevel)

			}
			logger.GetLogger(logName).SetLogLevel(logLevel)
			return c.String(http.StatusOK, fmt.Sprintf("Changed %s from %s to %s", logName, currentLevel, logLevel))
		} else {
			return c.String(http.StatusBadRequest, "logger name or level is missing")
		}
	})

	debug.GET("/properties", func(c echo.Context) error {
		props := ctx.Properties().SupportedProperties()
		data, _ := json.MarshalIndent(props, "", "  ")
		return c.Blob(200, echo.MIMEApplicationJSON, data)
	})

	debug.GET("/system/properties", func(c echo.Context) error {
		return c.JSON(200, properties.Global.GetAll())
	})

	debug.POST("/property", func(c echo.Context) error {
		name := c.Request().FormValue("name")
		value := c.Request().FormValue("value")
		if name == "" || value == "" {
			return c.String(http.StatusBadRequest, "property name or value is missing")
		}
		properties.Set(name, value)
		return c.NoContent(http.StatusOK)
	})

	debug.POST("/cron/run", func(c echo.Context) error {
		name := c.Request().FormValue("name")
		names := []string{}
		for entry := range Crons.IterBuffered() {
			for _, e := range entry.Val.Entries() {
				entry := toEntry(&e)
				names = append(names, entry.GetName())
				if entry.GetName() == name || fmt.Sprintf("%s/%s", entry.GetName(), entry.Context["id"]) == name {
					logger.Infof("Running %s now", name)
					e.Job.Run()
					return c.NoContent(http.StatusCreated)
				}
			}
		}
		return c.String(http.StatusNotFound, fmt.Sprintf("Cron job with name %s not found in %s", name, strings.Join(names, ", ")))

	})

	debug.GET("/cron", CronDetailsHandler())

	if period := properties.Duration(0, "memory.stats"); period > 0 {
		timer := timer.NewMemoryTimer()
		go func() {
			for {
				logger.GetLogger("memory").Infof("%s", timer.End())
				time.Sleep(period)
			}
		}()
	}
}

type JobCronEntry struct {
	Context   map[string]any `json:"context"`
	ID        string         `json:"id"`
	LastRan   time.Time      `json:"last_ran,omitempty"`
	NextRun   time.Time      `json:"next_run"`
	NextRunIn string         `json:"next_run_in"`
}

func (j JobCronEntry) GetName() string {
	if name, ok := j.Context["name"]; ok {
		return name.(string)
	}

	return ""
}

func toEntry(e *cron.Entry) JobCronEntry {
	entry := JobCronEntry{
		LastRan:   e.Prev,
		NextRun:   e.Next,
		NextRunIn: time.Until(e.Next).String(),
	}

	switch v := e.Job.(type) {
	case context.ContextAccessor:
		entry.Context = v.Context()
	case context.ContextAccessor2:
		entry.Context = v.GetContext()
	default:
		entry.Context = map[string]any{"name": fmt.Sprintf("%v", e.Job)}
	}

	if pk, ok := e.Job.(context.PKAccessor); ok {
		entry.ID = pk.PK()
	}
	return entry
}

func CronDetailsHandler() func(c echo.Context) error {
	return func(c echo.Context) error {
		var entries []JobCronEntry
		for entry := range Crons.IterBuffered() {
			for _, e := range entry.Val.Entries() {
				entries = append(entries, toEntry(&e))
			}
		}

		return c.JSON(http.StatusOK, entries)
	}
}
