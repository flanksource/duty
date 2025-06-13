package query

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flanksource/gomplate/v3/conv"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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
	var resourceSelectable dutyTypes.ResourceSelectable = dutyTypes.ResourceSelectableMap(resourceSelectableRaw)

	// NOTE: We check for fields in the map to determine what resource to unmarshal to.
	if _, ok := resourceSelectableRaw["config_class"]; ok {
		var config models.ConfigItem
		if err := config.FromMap(resourceSelectableRaw); err != nil {
			return false, fmt.Errorf("failed to unmarshal config item: %w", err)
		}

		resourceSelectable = config
	} else if _, ok := resourceSelectableRaw["category"]; ok {
		var playbook models.Playbook
		if b, err := json.Marshal(resourceSelectableRaw); err != nil {
			return false, err
		} else if err := json.Unmarshal(b, &playbook); err != nil {
			return false, err
		}

		resourceSelectable = &playbook
	} else if _, ok := resourceSelectableRaw["topology_id"]; ok {
		var component models.Component
		if b, err := json.Marshal(resourceSelectableRaw); err != nil {
			return false, err
		} else if err := json.Unmarshal(b, &component); err != nil {
			return false, err
		}

		resourceSelectable = component
	} else if _, ok := resourceSelectableRaw["canary_id"]; ok {
		var check models.Check
		if b, err := json.Marshal(resourceSelectableRaw); err != nil {
			return false, err
		} else if err := json.Unmarshal(b, &check); err != nil {
			return false, err
		}

		resourceSelectable = check
	}

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
