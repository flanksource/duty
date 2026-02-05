package echo

import (
	"net/http"

	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	echov4 "github.com/labstack/echo/v4"
)

func Properties(c echov4.Context) error {
	ctx := c.Request().Context().(context.Context)

	var dbProperties []models.AppProperty
	if err := ctx.DB().Find(&dbProperties).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, ctx.Oops().Wrap(err))
	}

	var seen = make(map[string]bool)

	var output = make([]map[string]string, 0)

	for k, v := range properties.Global.GetAll() {
		output = append(output, map[string]string{
			"name":        k,
			"value":       v,
			"source":      "local",
			"type":        "",
			"description": "",
		})
		seen[k] = true
	}

	for _, p := range dbProperties {
		if _, ok := seen[p.Name]; ok {
			continue
		}

		output = append(output, map[string]string{
			"name":        p.Name,
			"value":       p.Value,
			"source":      "db",
			"type":        "",
			"description": "",
		})
	}

	return c.JSON(http.StatusOK, output)
}
