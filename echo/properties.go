package echo

import (
	"net/http"
	"strings"

	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	echov4 "github.com/labstack/echo/v4"
)

// Properties returns all properties and must be protected by the caller's property-management authorization.
func Properties(c echov4.Context) error {
	return serveProperties(c, false)
}

// FeatureFlags returns only properties safe to expose to every signed-in user.
func FeatureFlags(c echov4.Context) error {
	return serveProperties(c, true)
}

func serveProperties(c echov4.Context, publicOnly bool) error {
	ctx := c.Request().Context().(context.Context)

	query := ctx.DB().Model(&models.AppProperty{})
	if publicOnly {
		query = query.Table("feature_flags")
	}
	var dbProperties []models.AppProperty
	if err := query.Find(&dbProperties).Error; err != nil {
		return api.WriteError(c, ctx.Oops().Wrap(err))
	}

	var seen = make(map[string]bool)

	var output = make([]map[string]string, 0)

	for name, value := range properties.Global.GetAll() {
		if publicOnly && !strings.HasPrefix(name, "public.") {
			continue
		}
		output = append(output, map[string]string{
			"name":        name,
			"value":       value,
			"source":      "local",
			"type":        "",
			"description": "",
		})
		seen[name] = true
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
