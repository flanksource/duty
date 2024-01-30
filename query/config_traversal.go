package query

import (
	"fmt"
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func TraverseConfig(ctx context.Context, id, relationType string) (models.ConfigItem, error) {
	var configItem models.ConfigItem

	relationTypeList := strings.Split(relationType, "/")

	for _, relType := range relationTypeList {
		configIDs, err := ConfigIDsByTypeFromCache(ctx, id, relType)
		if err != nil || len(configIDs) == 0 {
			return configItem, fmt.Errorf("no related type %s exists for config[%s]: %w", relType, id, err)
		}

		configID := configIDs[0]
		configItem, err = ConfigItemFromCache(ctx, configID)
		if err != nil {
			return configItem, fmt.Errorf("no config[%s] found in cache: %w", configID, err)
		}

		// Updating for next loop iteration
		id = configItem.ID.String()
	}

	return configItem, nil
}
