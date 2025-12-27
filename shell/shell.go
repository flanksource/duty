package shell

import (
	"bytes"
	gocontext "context"
	"fmt"
	"io"
	"maps"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/flanksource/artifacts"
	fileUtils "github.com/flanksource/commons/files"
	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/commons/utils"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/samber/oops"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
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
	Setup       *ExecSetup
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
	cmdCtx, err := prepareEnvironment(ctx, exec)
	if err != nil {
		return nil, ctx.Oops().Wrap(err)
	}

	envs := getEnvVar(cmdCtx.envs)
	runID := uuid.New().String()

	// PATH must be finalized before resolving the interpreter so /usr/bin/env uses the venv/runtime PATH.
	envs, err = applySetupRuntimeEnv(exec, envs)
	if err != nil {
		return nil, ctx.Oops().Wrap(err)
	}

	interpreter, _ := DetectInterpreterFromShebang(exec.Script)

	if exec.Setup != nil && exec.Setup.Python != nil && isPythonInterpreter(interpreter) {
		scriptPath, err := writeScriptToFile(runID, "python", "script.py", exec.Script)
		if err != nil {
			return nil, ctx.Oops().Wrap(err)
		}
		uvArgs := []string{"run", "--quiet"}
		if version := strings.TrimSpace(exec.Setup.Python.Version); version != "" {
			uvArgs = append(uvArgs, "--python", version)
		}
		uvArgs = append(uvArgs, scriptPath)
		uvPath, err := resolveInterpreterPath("uv", envs)
		if err != nil {
			return nil, ctx.Oops().Wrap(err)
		}
		cmd := osExec.CommandContext(ctx, uvPath, uvArgs...)
		cmd.Env = envs
		return runPreparedCmd(ctx, exec, cmd, cmdCtx, envs)
	}

	cmd, err := CreateCommandFromScript(ctx, exec.Script, envs)
	if err != nil {
		return nil, oops.Hint(exec.Script).Wrap(err)
	}

	return runPreparedCmd(ctx, exec, cmd, cmdCtx, envs)
}

func RunCmd(ctx context.Context, exec Exec, cmd *osExec.Cmd) (*ExecDetails, error) {
	ctx.Logger.V(3).Infof("running: %s %s", cmd.Path, lo.Map(cmd.Args, func(arg string, _ int) string { return strings.TrimSpace(arg) }))
	cmdCtx, err := prepareEnvironment(ctx, exec)
	if err != nil {
		return nil, ctx.Oops().Wrap(err)
	}

	envs := getEnvVar(cmdCtx.envs)

	envs, err = applySetupRuntimeEnv(exec, envs)
	if err != nil {
		return nil, ctx.Oops().Wrap(err)
	}

	return runPreparedCmd(ctx, exec, cmd, cmdCtx, envs)
}

func runPreparedCmd(ctx context.Context, exec Exec, cmd *osExec.Cmd, cmdCtx *commandContext, envs []string) (*ExecDetails, error) {
	cmd.Env = envs

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

	cmdCtx.artifacts = exec.Artifacts
	cmdCtx.cmd = cmd

	return runCmd(ctx, cmdCtx)
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

		result.mountPoint = filepath.Join(result.mountPoint, "exec-checkout", hash.Sha256Hex(checkout.URL))

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
			maps.Copy(result.extra, extra)
		}
	}

	return &result, nil
}

func getEnvVar(userSuppliedEnvs []string) []string {
	// Set to a non-nil empty slice to prevent access to current environment variables
	env := []string{}

	// Before the env vars from the host, because if there are duplicates
	// we use the first Env var that we see
	if len(userSuppliedEnvs) != 0 {
		env = append(env, userSuppliedEnvs...)
	}

	for _, e := range os.Environ() {
		key, _, ok := strings.Cut(e, "=")
		if _, exists := allowedEnvVars[key]; exists && ok {
			env = append(env, e)
		}
	}

	return env
}

func writeScriptToFile(runID string, runtime string, fileName string, script string) (string, error) {
	baseDir, err := resolveSetupBaseDir()
	if err != nil {
		return "", err
	}
	if baseDir == "" {
		return "", fmt.Errorf("shell-bin-dir is required for script file execution")
	}

	scriptDir := filepath.Join(baseDir, "runs", runID, runtime)
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return "", err
	}

	scriptPath := filepath.Join(scriptDir, fileName)
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return "", err
	}

	return scriptPath, nil
}
