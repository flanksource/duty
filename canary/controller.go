package canary

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
)

type SummaryResponse struct {
	Duration      int                   `json:"duration,omitempty"`
	RunnerName    string                `json:"runnerName"`
	ChecksSummary []models.CheckSummary `json:"checks_summary,omitempty"`
}

func SummaryHandler(c echo.Context) error {
	ctx := c.Request().Context().(context.Context)

	var queryOpt query.CheckSummaryOptions
	if err := c.Bind(&queryOpt); err != nil {
		return api.WriteError(c, api.Errorf(api.EINVALID, "invalid request: %v", err))
	}

	start := time.Now()
	results, err := query.CheckSummary(ctx, query.CheckSummaryOptions(queryOpt))
	if err != nil {
		return api.WriteError(c, err)
	}

	apiResponse := &SummaryResponse{
		RunnerName:    "local", // TODO: We don't have runnerName in here that canary-checker users
		ChecksSummary: results,
		Duration:      int(time.Since(start).Milliseconds()),
	}
	return c.JSON(http.StatusOK, apiResponse)
}
