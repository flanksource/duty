package query

import (
	"fmt"
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
)

func TraverseConfig(ctx context.Context, id, relationType string) (models.ConfigItem, error) {
	var configItem models.ConfigItem

	relationTypeList := strings.Split(relationType, "/")

	for _, relType := range relationTypeList {
		relatedIDs, err := ConfigRelationsFromCache(ctx, id)
		if err != nil {
			return configItem, fmt.Errorf("no relations found for config[%s] in cache: %w", id, err)
		}

		if len(relatedIDs) == 0 {
			return configItem, fmt.Errorf("no relations found for config[%s]: %w", id, err)
		}

		typeIDs, err := ConfigIDsByTypeFromCache(ctx, relType)
		if err != nil {
			return configItem, fmt.Errorf("no type %s exists for any config: %w", relType, err)
		}

		configID, ok := lo.Find(relatedIDs, func(relID string) bool {
			return lo.Contains(typeIDs, relID)
		})

		if !ok {
			return configItem, fmt.Errorf("no matching type %s found in relations for config[%s]", relType, id)
		}

		configItem, err = ConfigItemFromCache(ctx, configID)
		if err != nil {
			return configItem, fmt.Errorf("no config[%s] found in cache: %w", configID, err)
		}

		// Updating for next loop iteration
		id = configItem.ID.String()
	}

	return configItem, nil
}
