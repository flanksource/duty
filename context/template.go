package context

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/cel-go/cel"
	"github.com/samber/lo"
)

var CelEnvFuncs = make(map[string]func(Context) cel.EnvOption)
var TemplateFuncs = make(map[string]func(Context) any)

func (k Context) RunTemplate(t gomplate.Template, env map[string]any) (string, error) {
	l := k.Logger.Named("template")
	if l.V(3).Enabled() {
		l.V(3).Infof("Running template: %s with environment: %v", t.String(), logger.Pretty(env))
	} else if l.IsLevelEnabled(logger.Trace) {
		l.V(2).Infof("Running template: %s with environment keys: %v", t.String(), lo.Keys(env))
	} else {
		l.V(1).Infof("Running template: %s", t.String())
	}
	appendReferencedCelEnvFuncs(k, &t)
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
			if t.Template == val && l.V(4).Enabled() {
				l.V(4).Infof("%s = <no change>", t.String())
			} else if t.Template != val {
				if l.V(2).Enabled() {
					l.V(2).Infof("%s = %s", t.String(), val)
				} else if l.V(1).Enabled() {
					l.V(1).Infof("templated %s = changed", t.String())
				}
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

func appendReferencedCelEnvFuncs(ctx Context, t *gomplate.Template) {
	if t.Expression == "" || !strings.Contains(t.Expression, "(") {
		return
	}

	for name, f := range CelEnvFuncs {
		if celExpressionCalls(t.Expression, name) || (strings.HasSuffix(name, "Cel") && celExpressionCalls(t.Expression, strings.TrimSuffix(name, "Cel"))) {
			t.CelEnvs = append(t.CelEnvs, f(ctx))
		}
	}
}

func celExpressionCalls(expr, name string) bool {
	if name == "" {
		return false
	}

	for offset := 0; offset < len(expr); {
		idx := strings.Index(expr[offset:], name)
		if idx < 0 {
			return false
		}
		idx += offset
		afterName := idx + len(name)
		if isCelNameBoundary(expr, idx, afterName) {
			for afterName < len(expr) && isCELWhitespace(expr[afterName]) {
				afterName++
			}
			if afterName < len(expr) && expr[afterName] == '(' {
				return true
			}
		}
		offset = idx + len(name)
	}
	return false
}

func isCelNameBoundary(expr string, start, end int) bool {
	if start > 0 && isCELNameChar(expr[start-1]) {
		return false
	}
	if end < len(expr) && isCELNameChar(expr[end]) {
		return false
	}
	return true
}

func isCELNameChar(ch byte) bool {
	return ch == '.' || ch == '_' || ('0' <= ch && ch <= '9') || ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z')
}

func isCELWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
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
