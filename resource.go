package duty

import (
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
	"github.com/flanksource/gomplate/v3"
)

// GetResourceContext returns a common set of variables to be used in templates and expressions
func GetResourceContext(ctx context.Context, resource types.ResourceSelectable) map[string]any {
	output := map[string]any{}

	// Placeholders for commonly used fields of the resource
	// If a resource exists, they'll be filled up below
	output["name"] = ""
	output["status"] = ""
	output["health"] = ""
	output["labels"] = map[string]string{}
	output["tags"] = map[string]string{}

	tags := map[string]string{}
	if resource != nil {
		// set the alias name/status/health/labels/tags of the resource
		output["name"] = resource.GetName()
		if status, err := resource.GetStatus(); err == nil {
			output["status"] = status
		}
		if health, err := resource.GetHealth(); err == nil {
			output["health"] = health
		}
		if table, ok := resource.(models.TaggableModel); ok {
			if tableTags := table.GetTags(); tableTags != nil {
				tags = tableTags
			}
		}
		if table, ok := resource.(models.LabelableModel); ok {
			output["labels"] = table.GetLabels()
		}
	}

	if ctx.DB() != nil {
		if distinctTags, err := query.GetDistinctTags(ctx); err != nil {
			logger.Errorf("failed to get distinct tags for notification cel variable: %v", err)
		} else {
			for _, tag := range distinctTags {
				if _, ok := tags[tag]; !ok {
					tags[tag] = ""
				}
			}
		}
	}

	output["tags"] = tags

	// Inject tags as top level variables
	for k, v := range tags {
		if !gomplate.IsValidCELIdentifier(k) {
			logger.V(9).Infof("skipping tag %s as it is not a valid CEL identifier", k)
			continue
		}

		if _, ok := output[k]; ok {
			logger.V(9).Infof("skipping tag %s as it already exists in the playbook template environment", k)
			continue
		}

		output[k] = v
	}

	return output
}
