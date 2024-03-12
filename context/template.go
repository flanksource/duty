package context

import (
	"github.com/flanksource/commons/collections"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/cel-go/cel"
)

var CelEnvFuncs = make(map[string]func(Context) cel.EnvOption)
var TemplateFuncs = make(map[string]func(Context) any)

func (k Context) RunTemplate(t gomplate.Template, env map[string]any) (string, error) {
	for _, f := range CelEnvFuncs {
		t.CelEnvs = append(t.CelEnvs, f(k))
	}
	return gomplate.RunTemplate(env, t)
}

func (k Context) NewStructTemplater(vals map[string]any, requiredTag string, funcs map[string]any) gomplate.StructTemplater {
	tfuncs := make(map[string]any)
	for key, f := range TemplateFuncs {
		tfuncs[key] = f(k)
	}

	return gomplate.StructTemplater{
		Values:         vals,
		ValueFunctions: true,
		Funcs:          collections.MergeMap(tfuncs, funcs),
		RequiredTag:    requiredTag,
		DelimSets: []gomplate.Delims{
			{Left: "{{", Right: "}}"},
			{Left: "$((", Right: "))"},
		},
	}
}
