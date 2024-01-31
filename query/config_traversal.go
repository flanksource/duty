package query

import (
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func TraverseConfig(ctx context.Context, id, relationType string) *models.ConfigItem {
	var configItem models.ConfigItem

	relationTypeList := strings.Split(relationType, "/")

	for _, relType := range relationTypeList {
		configIDs, err := ConfigIDsByTypeFromCache(ctx, id, relType)
		if err != nil || len(configIDs) == 0 {
			ctx.Tracef("no related type %s exists for config[%s]: %v", relType, id, err)
			return nil
		}

		configID := configIDs[0]
		configItem, err = ConfigItemFromCache(ctx, configID)
		if err != nil {
			ctx.Tracef("no config[%s] found in cache: %v", configID, err)
			return nil
		}

		// Updating for next loop iteration
		id = configItem.ID.String()
	}

	return &configItem
}
