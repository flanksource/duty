package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
)

var DefaultInterpreter string
var DefaultInterpreterArgs []string

func init() {
	DefaultInterpreter, DefaultInterpreterArgs = DetectDefaultInterpreter()
}

// CreateCommandFromScript creates an os/exec.Cmd from the script, using the interpreter specified in the shebang line if present.
func CreateCommandFromScript(ctx context.Context, script string, envs []string) (*exec.Cmd, error) {
	interpreter, args := DetectInterpreterFromShebang(script)
	script = TrimLine(script, "#!")
	if script == "" {
		return nil, ctx.Oops().Errorf("empty script")
	}
	args = append(args, script)

	resolved, err := resolveInterpreterPath(interpreter, envs)
	if err != nil {
		return nil, ctx.Oops().Wrap(err)
	}

	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Env = envs
	return cmd, nil
}

func TrimLine(lines string, prefix string) string {
	s := []string{}
	for _, line := range strings.Split(lines, "\n") {
		if !strings.HasPrefix(line, prefix) {
			s = append(s, line)
		}
	}
	return strings.Join(s, "\n")
}

// DetectInterpreterFromShebang reads the first line of the script to detect the interpreter from the shebang line.
func DetectInterpreterFromShebang(script string) (string, []string) {
	reader := strings.NewReader(script)
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		firstLine := scanner.Text()
		if strings.HasPrefix(firstLine, "#!") {
			parts := strings.Fields(strings.TrimSpace(firstLine[2:]))
			if len(parts) == 0 {
				return "", nil
			}

			interpreter := parts[0]
			args := parts[1:]
			base := filepath.Base(interpreter)

			if base == "env" && len(args) > 0 {
				interpreter = args[0]
				args = args[1:]
				base = filepath.Base(interpreter)
			}

	switch base {
	case "python", "python3":
		if !lo.Contains(args, "-c") {
			args = append(args, "-c")
		}
	case "node":
		if !lo.Contains(args, "-e") {
			args = append(args, "-e")
		}
	case "bun":
		if !lo.Contains(args, "-e") {
			args = append(args, "-e")
		}
	default:
		if len(args) == 0 {
			// No args, just interpreter and assume it supports the -c flag
			args = append(args, "-c")
				}
			}

			return interpreter, args
		}
	}
	return DefaultInterpreter, DefaultInterpreterArgs
}

func isPythonInterpreter(interpreter string) bool {
	switch filepath.Base(interpreter) {
	case "python", "python3":
		return true
	default:
		return false
	}
}

func isNodeInterpreter(interpreter string) bool {
	switch filepath.Base(interpreter) {
	case "node":
		return true
	default:
		return false
	}
}

func isBunInterpreter(interpreter string) bool {
	switch filepath.Base(interpreter) {
	case "bun":
		return true
	default:
		return false
	}
}

// DetectDefaultInterpreter detects the default interpreter based on the OS.
func DetectDefaultInterpreter() (string, []string) {
	switch runtime.GOOS {
	case "windows":
		// Check for PowerShell on Windows
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			return "pwsh.exe", []string{"-c"}
		}
		// Fallback to cmd if PowerShell is not found
		if _, err := exec.LookPath("cmd.exe"); err == nil {
			return "cmd.exe", []string{"-c"}
		}

	default:
		// Check for Bash on Unix-like systems
		if _, err := exec.LookPath("bash"); err == nil {
			return "bash", []string{"-c"}
		}
		// Fallback to sh if Bash is not found
		if _, err := exec.LookPath("sh"); err == nil {
			return "sh", []string{"-c"}
		}
	}
	return "", nil
}

func resolveInterpreterPath(interpreter string, envs []string) (string, error) {
	if interpreter == "" {
		return "", fmt.Errorf("empty interpreter")
	}
	if filepath.IsAbs(interpreter) || strings.ContainsAny(interpreter, string(os.PathSeparator)+"/") {
		return interpreter, nil
	}

	pathEnv := pluckPathEnv(envs)
	if pathEnv == "" {
		return exec.LookPath(interpreter)
	}

	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, interpreter)
		if isExecutable(path) {
			return path, nil
		}
		if runtime.GOOS == "windows" {
			for _, ext := range []string{".exe", ".cmd", ".bat"} {
				if isExecutable(path + ext) {
					return path + ext, nil
				}
			}
		}
	}

	return "", exec.ErrNotFound
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0111 != 0
}
