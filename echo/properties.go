package echo

import (
	"encoding/json"
	"net/http"

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

func serveProperties(c echov4.Context, featureFlagsOnly bool) error {
	ctx := c.Request().Context().(context.Context)

	query := ctx.DB().Model(&models.AppProperty{})
	if featureFlagsOnly {
		query = query.Table("feature_flags")
	}
	var dbProperties []models.AppProperty
	if err := query.Find(&dbProperties).Error; err != nil {
		return api.WriteError(c, ctx.Oops().Wrap(err))
	}

	local := properties.Global.GetAll()
	names := make([]string, 0, len(local))
	for name := range local {
		names = append(names, name)
	}
	localJSON, err := json.Marshal(names)
	if err != nil {
		return api.WriteError(c, ctx.Oops().Wrap(err))
	}
	var localProperties []models.AppProperty
	if err := ctx.DB().Raw(`SELECT name, property_type(name) AS type
		FROM jsonb_array_elements_text(?::jsonb) AS names(name)`, string(localJSON)).Scan(&localProperties).Error; err != nil {
		return api.WriteError(c, ctx.Oops().Wrap(err))
	}

	var seen = make(map[string]bool)

	var output = make([]map[string]string, 0)

	for _, p := range localProperties {
		if featureFlagsOnly && p.Type != "feature_flag" {
			continue
		}
		output = append(output, map[string]string{
			"name":        p.Name,
			"value":       local[p.Name],
			"source":      "local",
			"type":        p.Type,
			"description": "",
		})
		seen[p.Name] = true
	}

	for _, p := range dbProperties {
		if _, ok := seen[p.Name]; ok {
			continue
		}

		output = append(output, map[string]string{
			"name":        p.Name,
			"value":       p.Value,
			"source":      "db",
			"type":        p.Type,
			"description": "",
		})
	}

	return c.JSON(http.StatusOK, output)
}
