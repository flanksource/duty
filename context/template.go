package context

import (
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/cel-go/cel"
)

var CelEnvFuncs []func(Context) cel.EnvOption

func (k Context) RunTemplate(t gomplate.Template, env map[string]any) (string, error) {
	for _, f := range CelEnvFuncs {
		logger.Infof("YASH APPENED")
		t.CelEnvs = append(t.CelEnvs, f(k))
	}
	return gomplate.RunTemplate(env, t)
}
