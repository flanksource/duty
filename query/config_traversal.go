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

func TraverseConfig(ctx context.Context, id, relationType, direction string) []models.ConfigItem {
	var configItems []models.ConfigItem

	targetRelType := relationType
	relationTypes := strings.Split(relationType, "/")
	relMap := make(map[string]bool)

	q := ctx.DB().Table("related_configs(?, ?)", id, direction).Select("id", "depth", "type")
	if len(relationTypes) == 1 {
		q = q.Where("type = ?", relationTypes[0])
	} else {
		targetRelType = relationTypes[len(relationTypes)-1]
		for i, rt := range relationTypes {
			q = q.Or("type = ? AND depth = ?", rt, i+1)
			relMap[rt] = false
		}
	}

	var rows []struct {
		ID    string
		Type  string
		Depth int
	}
	if err := q.Scan(&rows).Error; err != nil {
		ctx.Tracef("error querying database for related_configs[%s]: %v", id, err)
		return nil
	}

	for _, row := range rows {
		// Mark the paths as touched
		if len(relationTypes) > 1 {
			relMap[row.Type] = true
		}

		if row.Type != targetRelType {
			continue
		}

		configItem, err := ConfigItemFromCache(ctx, row.ID)
		if err != nil {
			ctx.Tracef("no config[%s] found in cache: %v", row.ID, err)
			continue
		}
		configItems = append(configItems, configItem)
	}

	// Make sure all the paths have matched
	if len(relationTypes) > 1 {
		for _, v := range relMap {
			if !v {
				return nil
			}
		}
	}

	return configItems
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
		return func(id, relationType, direction string) []models.ConfigItem {
			return TraverseConfig(ctx, id, relationType, direction)
		}
	}
}

func init() {
	context.CelEnvFuncs["catalog.traverse"] = traverseConfigCELFunction()
	context.TemplateFuncs["catalog_traverse"] = traverseConfigTemplateFunction()
}
