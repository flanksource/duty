package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/flanksource/commons/properties"
	"github.com/flanksource/deps"
)

// +kubebuilder:object:generate=true
type ExecSetup struct {
	Bun    *RuntimeSetup `json:"bun,omitempty" yaml:"bun,omitempty"`
	Python *RuntimeSetup `json:"python,omitempty" yaml:"python,omitempty"`
}

// +kubebuilder:object:generate=true
type RuntimeSetup struct {
	Version  string   `json:"version,omitempty" yaml:"version,omitempty"`
	Packages []string `json:"packages,omitempty" yaml:"packages,omitempty"`
}

// applySetupRuntimeEnv installs the dependencies and updates the PATH env var
func applySetupRuntimeEnv(exec Exec, envs []string) ([]string, error) {
	if exec.Setup == nil {
		return envs, nil
	}

	setupResult, err := installSetupRuntimes(*exec.Setup)
	if err != nil {
		return nil, err
	}

	if len(setupResult.binDirs) != 0 {
		envs = pathEnvWithBinDirs(envs, setupResult.binDirs)
	}

	envs, err = installSetupPackages(*exec.Setup, setupResult.baseDir, envs)
	if err != nil {
		return nil, err
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
	cacheDir   string
	tmpDir     string
}

func installSetupRuntimes(setup ExecSetup) (setupRuntimeResult, error) {
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
	}

	for _, runtime := range runtimes {
		if runtime.setup == nil {
			continue
		}
		version := strings.TrimSpace(runtime.setup.Version)
		if version == "" {
			version = "latest"
		}
		paths, err := installRuntime(runtime.name, version, baseDir)
		if err != nil {
			return result, err
		}

		result.binDirs = append(result.binDirs, paths.binDir)
	}

	return result, nil
}

func installSetupPackages(setup ExecSetup, baseDir string, envs []string) ([]string, error) {
	if baseDir == "" {
		return envs, nil
	}
	if setup.Python != nil && len(setup.Python.Packages) != 0 {
		return nil, fmt.Errorf("python packages are not installed via setup; use uv script dependencies")
	}
	if setup.Bun != nil && len(setup.Bun.Packages) != 0 {
		return nil, fmt.Errorf("bun packages are not installed via setup; use bun auto-install")
	}

	return envs, nil
}

func installRuntime(name string, version string, baseDir string) (runtimePaths, error) {
	runtimeDir := filepath.Join(baseDir, name, version)
	binDir := filepath.Join(runtimeDir, "bin")
	appDir := filepath.Join(runtimeDir, "apps")
	cacheDir := filepath.Join(runtimeDir, "cache")
	tmpDir := filepath.Join(runtimeDir, "tmp")

	for _, dir := range []string{binDir, appDir, cacheDir, tmpDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return runtimePaths{}, err
		}
	}

	_, err := deps.Install(name, version,
		deps.WithBinDir(binDir),
		deps.WithAppDir(appDir),
		deps.WithCacheDir(cacheDir),
		deps.WithTmpDir(tmpDir),
	)
	if err != nil {
		return runtimePaths{}, err
	}

	return runtimePaths{
		runtimeDir: runtimeDir,
		binDir:     binDir,
		appDir:     appDir,
		cacheDir:   cacheDir,
		tmpDir:     tmpDir,
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

func findRuntimeBinary(binDir string, candidates []string) (string, error) {
	suffixes := []string{""}
	if runtime.GOOS == "windows" {
		suffixes = append(suffixes, ".exe", ".cmd", ".bat")
	}

	for _, candidate := range candidates {
		for _, suffix := range suffixes {
			path := filepath.Join(binDir, candidate+suffix)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no runtime binary found in %s for %s", binDir, strings.Join(candidates, ", "))
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

func prependEnvVars(envs []string, vars map[string]string) []string {
	if len(vars) == 0 {
		return envs
	}

	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	updated := make([]string, 0, len(envs)+len(vars))
	for _, key := range keys {
		updated = append(updated, fmt.Sprintf("%s=%s", key, vars[key]))
	}

	for _, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if ok {
			if _, exists := vars[key]; exists {
				continue
			}
		}
		updated = append(updated, env)
	}

	return updated
}
