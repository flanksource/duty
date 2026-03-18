package query

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	dutyTypes "github.com/flanksource/duty/types"
	"github.com/flanksource/gomplate/v3/conv"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func CatalogQuery(ctx context.Context, selector string) []models.ConfigItem {
	rs := dutyTypes.ResourceSelector{Search: selector}
	items, err := FindConfigsByResourceSelector(ctx, 0, rs)
	if err != nil {
		ctx.Tracef("catalog.query error: %v", err)
		return nil
	}

	return items
}

func CatalogQueryOne(ctx context.Context, selector string) *models.ConfigItem {
	items := CatalogQuery(ctx, selector)
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func catalogQueryCELFunc() func(ctx context.Context) cel.EnvOption {
	return func(ctx context.Context) cel.EnvOption {
		return cel.Function("catalog.query",
			cel.Overload("catalog.query_string",
				[]*cel.Type{cel.StringType},
				cel.AnyType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					items := CatalogQuery(ctx, conv.ToString(args[0]))
					jsonObj, _ := conv.AnyToListMapStringAny(items)
					return types.NewDynamicList(types.DefaultTypeAdapter, jsonObj)
				}),
			),
			cel.Overload("catalog.query_string_list",
				[]*cel.Type{cel.StringType, cel.DynType},
				cel.AnyType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					selector := fmt.Sprintf(conv.ToString(args[0]), celListToAnySlice(args[1])...)
					items := CatalogQuery(ctx, selector)
					jsonObj, _ := conv.AnyToListMapStringAny(items)
					return types.NewDynamicList(types.DefaultTypeAdapter, jsonObj)
				}),
			),
		)
	}
}

func catalogQueryOneCELFunc() func(ctx context.Context) cel.EnvOption {
	return func(ctx context.Context) cel.EnvOption {
		return cel.Function("catalog.query_one",
			cel.Overload("catalog.query_one_string",
				[]*cel.Type{cel.StringType},
				cel.AnyType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					item := CatalogQueryOne(ctx, conv.ToString(args[0]))
					if item == nil {
						return types.NullValue
					}
					return types.DefaultTypeAdapter.NativeToValue(item.AsMap())
				}),
			),
			cel.Overload("catalog.query_one_string_list",
				[]*cel.Type{cel.StringType, cel.DynType},
				cel.AnyType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					selector := fmt.Sprintf(conv.ToString(args[0]), celListToAnySlice(args[1])...)
					item := CatalogQueryOne(ctx, selector)
					if item == nil {
						return types.NullValue
					}
					return types.DefaultTypeAdapter.NativeToValue(item.AsMap())
				}),
			),
		)
	}
}

// celListToAnySlice converts a CEL list ref.Val to []any for use as fmt.Sprintf args.
func celListToAnySlice(v ref.Val) []any {
	lv, ok := v.(traits.Lister)
	if !ok {
		return nil
	}
	iter := lv.Iterator()
	var result []any
	for iter.HasNext() == types.True {
		result = append(result, conv.ToString(iter.Next()))
	}
	return result
}

func catalogQueryTemplateFunc() func(ctx context.Context) any {
	return func(ctx context.Context) any {
		return func(selector string, args ...any) []models.ConfigItem {
			if len(args) > 0 {
				selector = fmt.Sprintf(selector, args...) //nolint:govet
			}
			return CatalogQuery(ctx, selector)
		}
	}
}

func catalogQueryOneTemplateFunc() func(ctx context.Context) any {
	return func(ctx context.Context) any {
		return func(selector string, args ...any) *models.ConfigItem {
			if len(args) > 0 {
				selector = fmt.Sprintf(selector, args...) //nolint:govet
			}
			return CatalogQueryOne(ctx, selector)
		}
	}
}

func init() {
	context.CelEnvFuncs["catalog.query"] = catalogQueryCELFunc()
	context.CelEnvFuncs["catalog.query_one"] = catalogQueryOneCELFunc()
	context.TemplateFuncs["catalog_query"] = catalogQueryTemplateFunc()
	context.TemplateFuncs["catalog_query_one"] = catalogQueryOneTemplateFunc()
}
