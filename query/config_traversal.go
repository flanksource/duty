package query

import (
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/gomplate/v3/conv"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func TraverseConfig(ctx context.Context, id, relationType, direction string) []models.ConfigItem {
	var configItems []models.ConfigItem

	//relationTypeList := strings.Split(relationType, "/")

	//q := `select id, type, depth from related_configs('e4051525-94ef-4cba-acee-6e44e34225ff'::uuid, ?) where type = ?`
	q := `select id, type, depth from related_configs(?, ?) where type = ?`
	var rows []struct {
		ID    string
		Type  string
		Depth int
	}
	if err := ctx.DB().Raw(q, id, direction, relationType).Scan(&rows).Error; err != nil {
		return nil
	}

	for _, row := range rows {
		configItem, err := ConfigItemFromCache(ctx, row.ID)
		if err != nil {
			ctx.Tracef("no config[%s] found in cache: %v", row.ID, err)
			continue
		}
		configItems = append(configItems, configItem)
	}

	return configItems
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
					items := TraverseConfig(ctx, configID, traversePath, "incoming")
					logger.Infof("GOT ITEMS %v", items)
					jsonObj, _ := conv.AnyToListMapStringAny(items)
					return types.NewDynamicList(types.DefaultTypeAdapter, jsonObj)
				}),
			),
		)
	}
}

func traverseConfigTemplateFunction() func(ctx context.Context) any {
	return func(ctx context.Context) any {
		return func(id, relationType string) []models.ConfigItem {
			return TraverseConfig(ctx, id, relationType, "incoming")
		}
	}
}

func init() {
	context.CelEnvFuncs["catalog.traverse"] = traverseConfigCELFunction()
	context.TemplateFuncs["catalog_traverse"] = traverseConfigTemplateFunction()
}
