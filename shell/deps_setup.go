package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flanksource/commons/properties"
	"github.com/flanksource/deps"
	"github.com/flanksource/duty/context"
)

// +kubebuilder:object:generate=true
type ExecSetup struct {
	Bun        *RuntimeSetup `json:"bun,omitempty" yaml:"bun,omitempty"`
	Python     *RuntimeSetup `json:"python,omitempty" yaml:"python,omitempty"`
	Powershell *RuntimeSetup `json:"powershell,omitempty" yaml:"powershell,omitempty"`
}

// +kubebuilder:object:generate=true
type RuntimeSetup struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// applySetupRuntimeEnv installs the dependencies and updates the PATH env var
func applySetupRuntimeEnv(ctx context.Context, setup ExecSetup, envs []string) ([]string, error) {
	setupResult, err := installSetupRuntimes(ctx, setup)
	if err != nil {
		return nil, err
	}

	if len(setupResult.binDirs) != 0 {
		envs = pathEnvWithBinDirs(envs, setupResult.binDirs)
	}

	return envs, nil
}

type setupRuntimeResult struct {
	binDirs []string
	baseDir string
}

type runtimePaths struct {
	runtimeDir string
	binDir     string
	appDir     string
}

func installSetupRuntimes(ctx context.Context, setup ExecSetup) (setupRuntimeResult, error) {
	result := setupRuntimeResult{}
	baseDir, err := resolveSetupBaseDir()
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

	return result, nil
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

func resolveSetupBaseDir() (string, error) {
	baseDir := properties.String("shell-bin-dir", "shell.bin.dir")
	if baseDir == "" {
		return "", nil
	}
	return filepath.Abs(baseDir)
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
