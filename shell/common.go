package shell

import (
	gocontext "context"
	"github.com/flanksource/duty/api"
	osExec "os/exec"
	"time"

	"github.com/flanksource/commons/properties"

	"github.com/flanksource/duty/context"
)

func JQ(ctx context.Context, path string, script string) (string, error) {
	_ctx, cancel := gocontext.WithTimeout(ctx, properties.Duration(5*time.Second, api.PropertyShellJQTimeout))
	defer cancel()

	cmd := osExec.CommandContext(_ctx, "jq", script, path)
	cmd.Env = getEnvVar(nil)
	result, err := runCmd(ctx, &commandContext{
		cmd: cmd,
	})
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func YQ(ctx context.Context, path string, script string) (string, error) {
	_ctx, cancel := gocontext.WithTimeout(ctx, properties.Duration(5*time.Second, api.PropertyShellYQTimeout, api.PropertyShellJQTimeout))
	defer cancel()

	cmd := osExec.CommandContext(_ctx, "yq", script, path)
	cmd.Env = getEnvVar(nil)
	result, err := runCmd(ctx, &commandContext{
		cmd: cmd,
	})
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
