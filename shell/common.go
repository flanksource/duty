package shell

import (
	gocontext "context"
	osExec "os/exec"
	"time"

	"github.com/flanksource/commons/properties"

	"github.com/flanksource/duty/context"
)

func JQ(ctx context.Context, path string, script string) (string, error) {
	_ctx, cancel := gocontext.WithTimeout(ctx, properties.Duration(5*time.Second, "shell.jq.timeout"))
	defer cancel()

	cmd := osExec.CommandContext(_ctx, "jq", script, path)
	result, err := RunCmd(ctx, Exec{
		Chroot: path,
	}, cmd)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func YQ(ctx context.Context, path string, script string) (string, error) {
	_ctx, cancel := gocontext.WithTimeout(ctx, properties.Duration(5*time.Second, "shell.yq.timeout", "shell.jq.timeout"))
	defer cancel()

	cmd := osExec.CommandContext(_ctx, "yq", script, path)
	result, err := RunCmd(ctx, Exec{
		Chroot: path,
	}, cmd)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
