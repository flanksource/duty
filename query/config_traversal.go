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

func TraverseConfig(ctx context.Context, configID, relationType, direction string) []models.ConfigItem {
	var configItems []models.ConfigItem

	ids := []string{configID}
	for _, typ := range strings.Split(relationType, "/") {
		ids = getRelatedTypeConfigID(ctx, ids, typ, direction, true)
	}

	for _, id := range ids {
		configItem, err := ConfigItemFromCache(ctx, id)
		if err != nil {
			ctx.Tracef("no config[%s] found in cache: %v", id, err)
			continue
		}
		configItems = append(configItems, configItem)
	}

	return configItems
}

// Fetch config IDs which match the type and direction upto max depth (5)
func getRelatedTypeConfigID(ctx context.Context, ids []string, relatedType, direction string, excludeSelf bool) []string {
	var allIDs []string
	for _, id := range ids {
		var rows []string
		q := ctx.DB().Table("related_configs_recursive(?, ?)", id, direction).Select("id").Where("type = ?", relatedType)
		if err := q.Scan(&rows).Error; err != nil {
			ctx.Tracef("error querying database for related_configs[%s]: %v", id, err)
			return nil
		}

		for _, row := range rows {
			if excludeSelf && row == id {
				continue
			}

			allIDs = append(allIDs, row)
		}
	}

	return allIDs
}

func traverseConfigCELFunction() func(ctx context.Context) cel.EnvOption {
	return func(ctx context.Context) cel.EnvOption {
		return cel.Function("catalog.traverse",
			cel.Overload("catalog.traverse_string_string",
				[]*cel.Type{cel.StringType, cel.StringType, cel.StringType},
				cel.AnyType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					if len(args) < 2 || len(args) > 3 {
						return types.String("invalid number of args")
					}
					id := conv.ToString(args[0])
					typ := conv.ToString(args[1])
					direction := "incoming"
					if len(args) == 3 {
						direction = conv.ToString(args[2])
					}
					items := TraverseConfig(ctx, id, typ, direction)
					jsonObj, _ := conv.AnyToListMapStringAny(items)
					return types.NewDynamicList(types.DefaultTypeAdapter, jsonObj)
				}),
			),
		)
	}
}

func traverseConfigTemplateFunction() func(ctx context.Context) any {
	return func(ctx context.Context) any {
		return func(args ...string) []models.ConfigItem {
			if len(args) < 2 {
				return nil
			}
			var id, relationType, direction string

			id = args[0]
			relationType = args[1]
			if len(args) == 3 {
				direction = args[2]
			} else {
				direction = "incoming"
			}
			return TraverseConfig(ctx, id, relationType, direction)
		}
	}
}

func init() {
	context.CelEnvFuncs["catalog.traverse"] = traverseConfigCELFunction()
	context.TemplateFuncs["catalog_traverse"] = traverseConfigTemplateFunction()
}
