package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/flanksource/commons/properties"
	"github.com/flanksource/deps"
)

// +kubebuilder:object:generate=true
type ExecSetup struct {
	Node   *RuntimeSetup `json:"node,omitempty" yaml:"node,omitempty"`
	Python *RuntimeSetup `json:"python,omitempty" yaml:"python,omitempty"`
}

// +kubebuilder:object:generate=true
type RuntimeSetup struct {
	Version  string   `json:"version,omitempty" yaml:"version,omitempty"`
	Packages []string `json:"packages,omitempty" yaml:"packages,omitempty"`
}

// applySetupRuntimeEnv installs the dependencies and updates the PATH env var
func applySetupRuntimeEnv(exec Exec, envs []string, runID string) ([]string, error) {
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

	envs, err = installSetupPackages(*exec.Setup, setupResult.baseDir, runID, envs)
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
		{name: "node", setup: setup.Node},
		{name: "python", setup: setup.Python},
	}

	for _, runtime := range runtimes {
		if runtime.setup == nil {
			continue
		}
		if runtime.setup.Version == "" {
			if len(runtime.setup.Packages) != 0 {
				return result, fmt.Errorf("%s packages requested but no version specified", runtime.name)
			}
			continue
		}
		paths, err := installRuntime(runtime.name, runtime.setup.Version, baseDir)
		if err != nil {
			return result, err
		}

		result.binDirs = append(result.binDirs, paths.binDir)
	}

	return result, nil
}

func installSetupPackages(setup ExecSetup, baseDir string, runID string, envs []string) ([]string, error) {
	if baseDir == "" {
		return envs, nil
	}

	runtimes := []struct {
		name  string
		setup *RuntimeSetup
	}{
		{name: "node", setup: setup.Node},
		{name: "python", setup: setup.Python},
	}

	for _, runtime := range runtimes {
		if runtime.setup == nil || len(runtime.setup.Packages) == 0 {
			continue
		}
		if runtime.setup.Version == "" {
			return nil, fmt.Errorf("%s packages requested but no version specified", runtime.name)
		}
		paths, err := installRuntime(runtime.name, runtime.setup.Version, baseDir)
		if err != nil {
			return nil, err
		}
		if err := installRuntimePackages(runtime.name, runtime.setup.Packages, paths, runID, baseDir); err != nil {
			return nil, err
		}
		switch runtime.name {
		case "node":
			envs = pathEnvWithBinDirs(envs, []string{filepath.Join(paths.appDir, "bin")})
			envs = prependEnvVars(envs, map[string]string{
				"NPM_CONFIG_PREFIX": paths.appDir,
			})
		case "python":
			venvDir := pythonVenvDir(baseDir, runID)
			envs = pathEnvWithBinDirs(envs, []string{pythonVenvBinDir(venvDir)})
			envs = prependEnvVars(envs, map[string]string{
				"VIRTUAL_ENV":      venvDir,
				"PYTHONNOUSERSITE": "1",
			})
		}
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

func installRuntimePackages(name string, packages []string, paths runtimePaths, runID string, baseDir string) error {
	if len(packages) == 0 {
		return nil
	}

	switch name {
	case "node":
		return installNodePackages(packages, paths)
	case "python":
		return installPythonPackages(packages, paths, runID, baseDir)
	default:
		return nil
	}
}

func installNodePackages(packages []string, paths runtimePaths) error {
	npmPath, err := findRuntimeBinary(paths.binDir, []string{"npm"})
	if err != nil {
		return err
	}

	args := append([]string{"install", "-g", "--prefix", paths.appDir}, packages...)
	cmd := exec.Command(npmPath, args...)
	cmd.Env = pathEnvWithBinDirs(os.Environ(), []string{paths.binDir})
	cmd.Env = prependEnvVars(cmd.Env, map[string]string{
		"NPM_CONFIG_PREFIX": paths.appDir,
	})

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func installPythonPackages(packages []string, paths runtimePaths, runID string, baseDir string) error {
	runtimePython, err := findRuntimeBinary(paths.binDir, []string{"python3", "python"})
	if err != nil {
		return err
	}

	venvDir := pythonVenvDir(baseDir, runID)
	if err := ensurePythonVenv(runtimePython, venvDir, paths.binDir); err != nil {
		return err
	}

	venvPython, err := findRuntimeBinary(pythonVenvBinDir(venvDir), []string{"python", "python3"})
	if err != nil {
		return err
	}

	args := append([]string{"-m", "pip", "install"}, packages...)
	cmd := exec.Command(venvPython, args...)
	cmd.Env = pathEnvWithBinDirs(os.Environ(), []string{pythonVenvBinDir(venvDir), paths.binDir})
	cmd.Env = prependEnvVars(cmd.Env, map[string]string{
		"VIRTUAL_ENV":      venvDir,
		"PYTHONNOUSERSITE": "1",
	})

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pip install failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	return nil
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

func pythonVenvDir(baseDir string, runID string) string {
	return filepath.Join(baseDir, "runs", runID, "python", "venv")
}

func pythonVenvBinDir(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts")
	}
	return filepath.Join(venvDir, "bin")
}

func ensurePythonVenv(runtimePython string, venvDir string, runtimeBinDir string) error {
	if _, err := os.Stat(venvDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(venvDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command(runtimePython, "-m", "venv", venvDir)
	cmd.Env = pathEnvWithBinDirs(os.Environ(), []string{runtimeBinDir})
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python venv failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	return nil
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
