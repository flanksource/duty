package query

import (
	"fmt"
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
)

//         - gitops:
//             repository:  "{{  catalog_traverse(.id, 'Kubernetes::Kustomization/Kubernetes::GitRepository').spec.repository }}"
//             file: "{{  catalog_traverse(.id, 'Kubernetes::Kustomization').spec.path }}/{{ .config.annotations['config.kubernetes.io/origin'] | YAML | .path }}"

// IN INIT (Setup cache via ID and type)SpecSpecSpecSpec

func TraverseConfig(ctx context.Context, id, relationType string) (models.ConfigItem, error) {
	var configItem models.ConfigItem

	relationTypeList := strings.Split(relationType, "/")
	relatedIDs, err := ConfigRelationsFromCache(ctx, id)
	if err != nil {
		return configItem, fmt.Errorf("no relations found for config[%s]: %w", id, err)
	}

	if len(relatedIDs) == 0 {
		return configItem, fmt.Errorf("no relations found for config[%s]: %w", id, err)
	}

	for _, relType := range relationTypeList {
		typeIDs, err := ConfigIDsByTypeFromCache(ctx, relType)
		if err != nil {
			return configItem, fmt.Errorf("no type %s exists for any config: %w", relType, err)
		}

		configID, ok := lo.Find(relatedIDs, func(relID string) bool {
			return lo.Contains(typeIDs, relID)
		})

		if !ok {
			return configItem, fmt.Errorf("no matching type %s found in relations for config[%s]: %w", relType, id, err)
		}

		configItem, err = ConfigItemFromCache(ctx, configID)
		if err != nil {
			return configItem, fmt.Errorf("no config[%s] found in cache: %w", configID, err)
		}
		// Updating to set correct error message
		id = configItem.ID.String()
	}

	return configItem, nil
}
