package query

import (
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/gomplate/v3/conv"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
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

func traverseConfigCELFunction() func(ctx context.Context) cel.EnvOption {
	return func(ctx context.Context) cel.EnvOption {
		return cel.Function("catalog.traverse",
			cel.Overload("catalog.traverse_string_string",
				[]*cel.Type{cel.StringType, cel.StringType},
				cel.AnyType,
				cel.BinaryBinding(func(id ref.Val, path ref.Val) ref.Val {
					configID := conv.ToString(id)
					traversePath := conv.ToString(path)
					item := TraverseConfig(ctx, configID, traversePath)
					jsonObj, _ := conv.AnyToMapStringAny(item)
					return types.NewDynamicMap(types.DefaultTypeAdapter, jsonObj)
				}),
			),
		)
	}
}

func init() {
	context.CelEnvFuncs = append(context.CelEnvFuncs, traverseConfigCELFunction())
}
