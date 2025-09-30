package topology

import (
	"net/http"

	echov4 "github.com/labstack/echo/v4"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"
)

func QueryHandler(c echov4.Context) error {
	ctx := c.Request().Context().(context.Context)
	params := query.NewTopologyParams(c.QueryParams())
	results, err := query.Topology(ctx, params)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(http.StatusOK, results)
}
