package job

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
)

type JobCronEntry struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Schedule     string    `json:"schedule"`
	ResourceID   string    `json:"resource_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	LastRan      time.Time `json:"last_ran,omitempty"`
	NextRun      time.Time `json:"next_run"`
	NextRunIn    string    `json:"next_run_in"`
}

func CronDetailsHandler(crons ...*cron.Cron) func(c echo.Context) error {
	return func(c echo.Context) error {
		var entries []cron.Entry
		for i := range crons {
			entries = append(entries, crons[i].Entries()...)
		}

		mapped := lo.Map(entries, func(e cron.Entry, _ int) JobCronEntry {
			j, ok := e.Job.(*Job)
			if !ok {
				return JobCronEntry{}
			}

			return JobCronEntry{
				ID:           j.ID,
				ResourceID:   j.ResourceID,
				ResourceType: j.ResourceType,
				Name:         j.Name,
				Schedule:     j.Schedule,
				LastRan:      e.Prev,
				NextRun:      e.Next,
				NextRunIn:    time.Until(e.Next).String(),
			}
		})

		return c.JSON(http.StatusOK, mapped)
	}
}
