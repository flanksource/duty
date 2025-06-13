package query

import (
	"errors"
	"fmt"

	"github.com/flanksource/gomplate/v3/conv"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/flanksource/duty/context"
	dutyTypes "github.com/flanksource/duty/types"
)

func init() {
	context.CelEnvFuncs["matchQueryCel"] = MatchQueryCelFunc
	context.TemplateFuncs["matchQueryTemplate"] = MatchQueryGotemplateFunc
}

func MatchQueryGotemplateFunc(ctx context.Context) any {
	return matchQuery
}

func MatchQueryCelFunc(ctx context.Context) cel.EnvOption {
	return cel.Function("matchQuery",
		cel.Overload("matchQuery",
			[]*cel.Type{cel.MapType(cel.StringType, cel.DynType), cel.StringType},
			cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				peg := conv.ToString(args[1])

				resourceSelectableRaw, err := convertMap(args[0])
				if err != nil {
					return types.WrapErr(errors.New("matchQuery expects the first argument to be a map[string]any"))
				}

				match, err := matchQuery(resourceSelectableRaw, peg)
				if err != nil {
					return types.WrapErr(fmt.Errorf("matchQuery failed: %w", err))
				}

				return types.Bool(match)
			}),
		),
	)
}

func matchQuery(resourceSelectableRaw map[string]any, peg string) (bool, error) {
	resourceSelectable := dutyTypes.ResourceSelectableMap(resourceSelectableRaw)
	rs := dutyTypes.ResourceSelector{Search: peg}
	return rs.Matches(resourceSelectable), nil
}

func convertMap(arg ref.Val) (map[string]any, error) {
	switch m := arg.Value().(type) {
	case map[ref.Val]ref.Val:
		var out = make(map[string]any)
		for key, val := range m {
			out[key.Value().(string)] = val.Value()
		}
		return out, nil
	case map[string]any:
		return m, nil
	default:
		return nil, fmt.Errorf("not a map %T", arg.Value())
	}
}
