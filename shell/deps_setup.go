package shell

import (
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"

	"github.com/flanksource/deps"
	"github.com/flanksource/duty/context"
)

// +kubebuilder:object:generate=true
type ExecSetup struct {
	Bun        *RuntimeSetup `json:"bun,omitempty" yaml:"bun,omitempty"`
	Python     *RuntimeSetup `json:"python,omitempty" yaml:"python,omitempty"`
	Powershell *RuntimeSetup `json:"powershell,omitempty" yaml:"powershell,omitempty"`
	Playwright *RuntimeSetup `json:"playwright,omitempty" yaml:"playwright,omitempty"`
}

// +kubebuilder:object:generate=true
type RuntimeSetup struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// applySetupRuntimeEnv installs the dependencies and updates the PATH env var
func applySetupRuntimeEnv(ctx context.Context, exec *Exec, envs []string) ([]string, error) {
	setupResult, err := installSetupRuntimes(ctx, exec)
	if err != nil {
		return nil, err
	}

	if len(setupResult.binDirs) != 0 {
		envs = pathEnvWithBinDirs(envs, setupResult.binDirs)
	}

	envs = append(envs, setupResult.extraEnv...)

	return envs, nil
}

type setupRuntimeResult struct {
	binDirs  []string
	baseDir  string
	extraEnv []string
}

type runtimePaths struct {
	runtimeDir string
	binDir     string
	appDir     string
}

func installSetupRuntimes(ctx context.Context, exec *Exec) (setupRuntimeResult, error) {
	result := setupRuntimeResult{}
	baseDir := exec.BaseDir
	setup := exec.Setup
	if baseDir == "" {
		baseDir = ".shell"
	}
	baseDir, err := filepath.Abs(baseDir)
	if err != nil || baseDir == "" {
		return result, err
	}
	result.baseDir = baseDir

	runtimes := []struct {
		name  string
		setup *RuntimeSetup
	}{
		{name: "bun", setup: setup.Bun},
		{name: "uv", setup: setup.Python},
		{name: "powershell", setup: setup.Powershell},
	}

	for _, runtime := range runtimes {
		if runtime.setup == nil {
			continue
		}

		version := strings.TrimSpace(runtime.setup.Version)
		if runtime.name == "uv" || version == "" {
			// we must not map the python version to the uv version.
			// We always download the latest uv version.
			version = "any"
		}

		paths, err := installRuntime(ctx, runtime.name, version, baseDir)
		if err != nil {
			return result, err
		}

		result.binDirs = append(result.binDirs, paths.binDir)
	}

	if setup.Playwright != nil {
		pwResult, err := installPlaywrightRuntime(ctx, setup.Playwright, baseDir)
		if err != nil {
			return result, fmt.Errorf("failed to install playwright runtime: %w", err)
		}
		result.binDirs = append(result.binDirs, pwResult.binDirs...)
		result.extraEnv = append(result.extraEnv, pwResult.extraEnv...)
	}

	return result, nil
}

func installPlaywrightRuntime(ctx context.Context, setup *RuntimeSetup, baseDir string) (setupRuntimeResult, error) {
	var nodeBinDir string

	nodePaths, err := installRuntime(ctx, "node", "any", baseDir)
	if err != nil {
		return setupRuntimeResult{}, fmt.Errorf("failed to install node for playwright: %w", err)
	}

	// deps may detect node as already installed on PATH.
	// Check if npm exists in the installed binDir; if not, find it on PATH.
	npmBin := filepath.Join(nodePaths.binDir, "npm")
	if _, err := os.Stat(npmBin); err != nil {
		if systemNpm, lookErr := osExec.LookPath("npm"); lookErr == nil {
			npmBin = systemNpm
			nodeBinDir = filepath.Dir(systemNpm)
		} else {
			return setupRuntimeResult{}, fmt.Errorf("npm not found in %s or on PATH", nodePaths.binDir)
		}
	} else {
		nodeBinDir = nodePaths.binDir
	}

	npxBin := filepath.Join(nodeBinDir, "npx")
	if _, err := os.Stat(npxBin); err != nil {
		if systemNpx, lookErr := osExec.LookPath("npx"); lookErr == nil {
			npxBin = systemNpx
		} else {
			return setupRuntimeResult{}, fmt.Errorf("npx not found in %s or on PATH", nodeBinDir)
		}
	}

	version := strings.TrimSpace(setup.Version)
	if version == "" {
		version = "latest"
	}

	playwrightDir := filepath.Join(baseDir, "playwright", version)
	browsersDir := filepath.Join(playwrightDir, "browsers")
	for _, dir := range []string{playwrightDir, browsersDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return setupRuntimeResult{}, fmt.Errorf("failed to create playwright directory %s: %w", dir, err)
		}
	}

	envWithPath := append(os.Environ(),
		fmt.Sprintf("PATH=%s%c%s", nodeBinDir, os.PathListSeparator, os.Getenv("PATH")),
		fmt.Sprintf("PLAYWRIGHT_BROWSERS_PATH=%s", browsersDir),
	)

	initCmd := osExec.CommandContext(ctx, npmBin, "init", "-y")
	initCmd.Dir = playwrightDir
	initCmd.Env = envWithPath
	if out, err := initCmd.CombinedOutput(); err != nil {
		return setupRuntimeResult{}, fmt.Errorf("npm init failed: %s: %w", string(out), err)
	}

	installCmd := osExec.CommandContext(ctx, npmBin, "install", "playwright@"+version)
	installCmd.Dir = playwrightDir
	installCmd.Env = envWithPath
	if out, err := installCmd.CombinedOutput(); err != nil {
		return setupRuntimeResult{}, fmt.Errorf("npm install playwright@%s failed: %s: %w", version, string(out), err)
	}

	browserCmd := osExec.CommandContext(ctx, npxBin, "playwright", "install", "chromium")
	browserCmd.Dir = playwrightDir
	browserCmd.Env = envWithPath
	if out, err := browserCmd.CombinedOutput(); err != nil {
		return setupRuntimeResult{}, fmt.Errorf("playwright install chromium failed: %s: %w", string(out), err)
	}

	nodeModules := filepath.Join(playwrightDir, "node_modules")
	return setupRuntimeResult{
		binDirs: []string{nodeBinDir, filepath.Join(nodeModules, ".bin")},
		extraEnv: []string{
			fmt.Sprintf("NODE_PATH=%s", nodeModules),
			fmt.Sprintf("PLAYWRIGHT_BROWSERS_PATH=%s", browsersDir),
		},
	}, nil
}

func installRuntime(ctx context.Context, name string, version string, baseDir string) (runtimePaths, error) {
	runtimeDir := filepath.Join(baseDir, name, version)
	binDir := filepath.Join(runtimeDir, "bin")
	appDir := filepath.Join(runtimeDir, "apps")

	for _, dir := range []string{binDir, appDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return runtimePaths{}, err
		}
	}

	result, err := deps.InstallWithContext(ctx, name, version,
		deps.WithBinDir(binDir),
		deps.WithAppDir(appDir),
	)
	if result != nil && ctx.IsDebug() {
		ctx.Debugf("%s", result.Pretty().ANSI())
	}
	if err != nil {
		return runtimePaths{}, err
	}

	return runtimePaths{
		runtimeDir: runtimeDir,
		binDir:     binDir,
		appDir:     appDir,
	}, nil
}

func pluckPathEnv(envs []string) string {
	for _, env := range envs {
		if after, ok := strings.CutPrefix(env, "PATH="); ok {
			return after
		}
	}

	return ""
}

func pathEnvWithBinDirs(envs []string, binDirs []string) []string {
	if len(binDirs) == 0 {
		return envs
	}

	newPath := make([]string, 0, len(binDirs))

	// Append the bin dirs to PATH
	for _, binDir := range binDirs {
		binDir = strings.TrimSpace(binDir)
		if binDir != "" {
			newPath = append(newPath, binDir)
		}
	}

	// Append the existing path after bin dirs
	existingPath := pluckPathEnv(envs)
	if existingPath != "" {
		newPath = append(newPath, existingPath)
	}

	for i, env := range envs {
		if strings.HasPrefix(env, "PATH=") {
			envs[i] = fmt.Sprintf("PATH=%s", strings.Join(newPath, string(os.PathListSeparator)))
			return envs
		}
	}

	envs = append(envs, fmt.Sprintf("PATH=%s", strings.Join(newPath, string(os.PathListSeparator))))
	return envs
}
