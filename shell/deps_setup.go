package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
func applySetupRuntimeEnv(exec Exec, cmd *exec.Cmd) error {
	if exec.Setup == nil {
		return nil
	}

	binDirs, err := installSetupRuntimes(*exec.Setup)
	if err != nil {
		return err
	} else if len(binDirs) == 0 {
		return nil
	}

	cmd.Env = pathEnvWithBinDirs(cmd.Env, binDirs)

	return nil
}

func installSetupRuntimes(setup ExecSetup) ([]string, error) {
	baseDir, err := resolveSetupBaseDir()
	if err != nil || baseDir == "" {
		return nil, err
	}

	runtimes := []struct {
		name  string
		setup *RuntimeSetup
	}{
		{name: "node", setup: setup.Node},
		{name: "python", setup: setup.Python},
	}

	binDirs := []string{}
	for _, runtime := range runtimes {
		if runtime.setup == nil || runtime.setup.Version == "" {
			continue
		}
		binDir, err := installRuntime(runtime.name, runtime.setup.Version, baseDir)
		if err != nil {
			return nil, err
		}
		if binDir != "" {
			binDirs = append(binDirs, binDir)
		}
	}

	return binDirs, nil
}

func installRuntime(name string, version string, baseDir string) (string, error) {
	runtimeDir := filepath.Join(baseDir, name, version)
	binDir := filepath.Join(runtimeDir, "bin")
	appDir := filepath.Join(runtimeDir, "apps")
	cacheDir := filepath.Join(runtimeDir, "cache")
	tmpDir := filepath.Join(runtimeDir, "tmp")

	for _, dir := range []string{binDir, appDir, cacheDir, tmpDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}

	_, err := deps.Install(name, version,
		deps.WithBinDir(binDir),
		deps.WithAppDir(appDir),
		deps.WithCacheDir(cacheDir),
		deps.WithTmpDir(tmpDir),
	)
	if err != nil {
		return "", err
	}

	return binDir, nil
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
		}
	}

	return envs
}
