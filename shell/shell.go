package shell

import (
	"bytes"
	gocontext "context"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/flanksource/artifacts"
	fileUtils "github.com/flanksource/commons/files"
	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

// List of env var keys that we pass on to the exec command
var allowedEnvVars = map[string]struct{}{
	"CLOUDSDK_PYTHON":                       {},
	"DEBIAN_FRONTEND":                       {},
	"DOTNET_SYSTEM_GLOBALIZATION_INVARIANT": {},
	"HOME":                                  {},
	"LC_CTYPE":                              {},
	"PATH":                                  {},
	"PS_INSTALL_FOLDER":                     {},
	"PS_VERSION":                            {},
	"PSModuleAnalysisCachePath":             {},
	"USER":                                  {},
	"MANPATH":                               {},
	"TERM":                                  {},
	"LANG":                                  {},
	"SHELL":                                 {},
	"SHLVL":                                 {},
	"LC_ALL":                                {},
	"JAVA_HOME":                             {},
	"SDKMAN_DIR":                            {},
	"LSCOLORS":                              {},
	"CLICOLOR":                              {},
	"COLORTERM":                             {},
	"TERM_PROGRAM":                          {},
	"TERM_PROGRAM_VERSION":                  {},
	"COLORFGBG":                             {},
}

func init() {
	for _, env := range strings.Split(properties.String("", "shell.allowed.envs"), ",") {
		logger.V(5).Infof("allowing env var %s", env)
		allowedEnvVars[env] = struct{}{}
	}
}

var checkoutLocks = utils.NamedLock{}

type Exec struct {
	Script      string
	Connections connection.ExecConnections
	Checkout    *connection.GitConnection
	Artifacts   []Artifact
	EnvVars     []types.EnvVar
	Chroot      string
}

// +kubebuilder:object:generate=true
type Artifact struct {
	Path string `json:"path" yaml:"path" template:"true"`
}

type ExecDetails struct {
	Stdout   string   `json:"stdout"`
	Stderr   string   `json:"stderr"`
	ExitCode int      `json:"exitCode"`
	Path     string   `json:"path"`
	Args     []string `json:"args"`

	// Any extra details about the command execution, e.g. git commit id, etc.
	Extra map[string]any `json:"extra,omitempty"`

	Error     error                `json:"-" yaml:"-"`
	Artifacts []artifacts.Artifact `json:"-" yaml:"-"`
}

func (e ExecDetails) String() string {
	return fmt.Sprintf("%s %s exit=%d stdout=%s stderr=%s", e.Path, e.Args, e.ExitCode, e.Stdout, e.Stderr)
}

func (e *ExecDetails) GetArtifacts() []artifacts.Artifact {
	if e == nil {
		return nil
	}
	return e.Artifacts
}

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

func Run(ctx context.Context, exec Exec) (*ExecDetails, error) {
	cmd, err := CreateCommandFromScript(ctx, exec.Script)
	if err != nil {
		return nil, oops.Hint(exec.Script).Wrap(err)
	}

	return RunCmd(ctx, exec, cmd)
}

func RunCmd(ctx context.Context, exec Exec, cmd *osExec.Cmd) (*ExecDetails, error) {
	ctx.Logger.V(3).Infof("running: %s %s", cmd.Path, lo.Map(cmd.Args, func(arg string, _ int) string { return strings.TrimSpace(arg) }))
	envParams, err := prepareEnvironment(ctx, exec)
	if err != nil {
		return nil, ctx.Oops().Wrap(err)
	}

	if exec.Chroot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, ctx.Oops().Wrap(err)
		}
		cmdDir := envParams.mountPoint
		if cmdDir == "" {
			cmdDir = path.Join(properties.String(cwd, "shell.tmp.dir"), "shell-tmp", uuid.New().String())
			if err := os.MkdirAll(cmdDir, 0700); err != nil {
				return nil, ctx.Oops().Wrap(err)
			}
		}
		cmd.Dir = cmdDir
	} else {
		if stat, err := os.Stat(exec.Chroot); err != nil {
			return nil, ctx.Oops().Wrap(err)
		} else if stat.IsDir() {
			envParams.mountPoint = stat.Name()
			return nil, fmt.Errorf("%s is not a directory", exec.Chroot)
		} else {
			envParams.mountPoint = filepath.Dir(stat.Name())
		}
		cmd.Dir = exec.Chroot
	}

	// Set to a non-nil empty slice to prevent access to current environment variables
	cmd.Env = []string{}

	for _, e := range os.Environ() {
		key, _, ok := strings.Cut(e, "=")
		if _, exists := allowedEnvVars[key]; exists && ok {
			cmd.Env = append(cmd.Env, e)
		}
	}

	if len(envParams.envs) != 0 {
		cmd.Env = append(cmd.Env, envParams.envs...)
	}

	if setupResult, err := connection.SetupConnection(ctx, exec.Connections, cmd); err != nil {
		return nil, ctx.Oops().Wrap(err)
	} else {
		ctx = ctx.WithLoggingValues("connection", setupResult)
		defer func() {
			if waitBeforeCleanup := ctx.Properties().Duration("shell.connection.wait_before_cleanup", 0); waitBeforeCleanup > 0 {
				time.Sleep(waitBeforeCleanup)
			}
			if err := setupResult.Cleanup(); err != nil {
				logger.Errorf("failed to cleanup connection artifacts: %v", err)
			}
		}()
	}

	envParams.artifacts = exec.Artifacts
	envParams.cmd = cmd

	return runCmd(ctx, envParams)
}

type commandContext struct {
	cmd       *osExec.Cmd
	artifacts []Artifact
	EnvVars   []types.EnvVar
	extra     map[string]any

	// Working directory for the command
	mountPoint string

	// Additional env vars to be exported into the shell
	envs []string
}

func runCmd(ctx context.Context, cmd *commandContext) (*ExecDetails, error) {
	var (
		result ExecDetails
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd.cmd.Stdout = &stdout
	cmd.cmd.Stderr = &stderr

	ctx.Logger.V(6).Infof("working directory: %s\nenvironment:\n%s", cmd.mountPoint, strings.Join(cmd.cmd.Env, "\n"))

	result.Error = cmd.cmd.Run()
	result.Args = cmd.cmd.Args
	result.Extra = cmd.extra
	result.Path = cmd.cmd.Path
	result.ExitCode = cmd.cmd.ProcessState.ExitCode()
	result.Stderr = strings.TrimSpace(stderr.String())
	result.Stdout = strings.TrimSpace(stdout.String())

	ctx.Logger.V(3).Infof("%s exited with code=%d, stdout=%d bytes, stderr=%d bytes", cmd.cmd.Path, result.ExitCode, len(result.Stdout), len(result.Stderr))

	for _, artifactConfig := range cmd.artifacts {
		switch artifactConfig.Path {
		case "/dev/stdout":
			result.Artifacts = append(result.Artifacts, artifacts.Artifact{
				Content: io.NopCloser(strings.NewReader(result.Stdout)),
				Path:    "stdout",
			})

		case "/dev/stderr":
			result.Artifacts = append(result.Artifacts, artifacts.Artifact{
				Content: io.NopCloser(strings.NewReader(result.Stderr)),
				Path:    "stderr",
			})

		default:
			paths, err := fileUtils.DoubleStarGlob(cmd.mountPoint, []string{artifactConfig.Path})
			if err != nil {
				return nil, err
			}

			for _, path := range paths {
				file, err := os.Open(path)
				if err != nil {
					return nil, fmt.Errorf("error opening artifact file. path=%s; %w", path, err)
				}

				if stat, err := file.Stat(); err != nil {
					return nil, fmt.Errorf("error getting artifact file stat. path=%s; %w", path, err)
				} else if stat.IsDir() {
					return nil, fmt.Errorf("artifact path (%s) is a directory. expected file", path)
				}

				result.Artifacts = append(result.Artifacts, artifacts.Artifact{
					Content: file,
					Path:    path,
				})
			}
		}
	}
	if result.ExitCode != 0 {
		return &result, ctx.Oops().With(
			"cmd", cmd.cmd.Path,
			"args", cmd.cmd.Args,
			"error", result.Error.Error(),
			"stderr", result.Stderr,
			"stdout", result.Stdout,
			"extra", result.Extra,
			"exit-code", result.ExitCode,
		).Code(fmt.Sprintf("exited with %d", result.ExitCode)).Errorf("%v", result.Error.Error())
	}

	return &result, nil
}

func prepareEnvironment(ctx context.Context, exec Exec) (*commandContext, error) {
	result := commandContext{
		extra: make(map[string]any),
	}

	for _, env := range exec.EnvVars {
		val, err := ctx.GetEnvValueFromCache(env, ctx.GetNamespace())
		if err != nil {
			return nil, fmt.Errorf("error fetching env value (name=%s): %w", env.Name, err)
		}

		result.envs = append(result.envs, fmt.Sprintf("%s=%s", env.Name, val))
	}

	if exec.Checkout != nil {
		checkout := *exec.Checkout

		if err := checkout.HydrateConnection(ctx); err != nil {
			return nil, fmt.Errorf("error hydrating connection: %w", err)
		}

		if dir := lo.FromPtr(checkout.Destination); dir != "" {
			result.mountPoint = filepath.Join(result.mountPoint, dir)
		} else {
			result.mountPoint = filepath.Join(result.mountPoint, "exec-checkout", hash.Sha256Hex(checkout.URL))
		}
		// We allow multiple checks to use the same checkout location, for disk space and performance reasons
		// however git does not allow multiple operations to be performed, so we need to lock it
		lock := checkoutLocks.TryLock(result.mountPoint, 5*time.Minute)
		if lock == nil {
			return nil, fmt.Errorf("failed to acquire checkout lock for %s", result.mountPoint)
		}
		defer lock.Release()

		client, err := connection.CreateGitConfig(ctx, &checkout)
		if err != nil {
			return nil, err
		}

		if extra, err := client.Clone(ctx, result.mountPoint); err != nil {
			return nil, err
		} else {
			for k, v := range extra {
				result.extra[k] = v
			}
		}
	}

	return &result, nil
}
