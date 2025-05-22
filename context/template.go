package context

import (
	"fmt"
	"strconv"

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
	if t.Functions == nil {
		t.Functions = make(map[string]any)
	}
	for name, v := range TemplateFuncs {
		t.Functions[name] = v(k)
	}

	if t.Template != "" {
		// For go templates, we try both [{{}}, $()] delimiters by default if no explicit delimiters are provided
		delimSets := []gomplate.Delims{
			{Left: "{{", Right: "}}"},
			{Left: "$(", Right: ")"},
		}

		if t.LeftDelim != "" || t.RightDelim != "" {
			delimSets = []gomplate.Delims{
				{Left: t.LeftDelim, Right: t.RightDelim},
			}
		}

		for _, delimSet := range delimSets {
			t.LeftDelim = delimSet.Left
			t.RightDelim = delimSet.Right

			val, err := gomplate.RunTemplateContext(k.Context, env, t)
			if err != nil {
				return "", k.Oops().With("template", t.String(), "environment", env).Wrap(err)
			}
			t.Template = val
		}

		return t.Template, nil
	}

	val, err := gomplate.RunTemplateContext(k.Context, env, t)
	if err != nil {
		return "", k.Oops().With("template", t.String(), "environment", env).Wrap(err)
	}

	return val, nil
}

func (k Context) RunTemplateBool(t gomplate.Template, env map[string]any) (bool, error) {
	output, err := k.RunTemplate(t, env)
	if err != nil {
		return false, err
	}

	result, err := strconv.ParseBool(output)
	if err != nil {
		return false, fmt.Errorf("failed to parse template output (%s) as bool: %w", output, err)
	}

	return result, nil
}

func (k Context) NewStructTemplater(vals map[string]any, requiredTag string, funcs map[string]any) gomplate.StructTemplater {
	tfuncs := make(map[string]any)
	for key, f := range TemplateFuncs {
		tfuncs[key] = f(k)
	}

	return gomplate.StructTemplater{
		Context:        k.Context,
		Values:         vals,
		ValueFunctions: true,
		Funcs:          collections.MergeMap(tfuncs, funcs),
		RequiredTag:    requiredTag,
		DelimSets: []gomplate.Delims{
			{Left: "{{", Right: "}}"},
			{Left: "$(", Right: ")"},
		},
	}
}
