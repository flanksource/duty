package shell

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

var _ = ginkgo.Describe("Shell Run", ginkgo.Label("slow"), func() {
	ginkgo.AfterEach(func() {
		os.RemoveAll("./shell-tmp/")
	})

	ctx := context.New()

	ginkgo.It("should run bun scripts", func() {
		exec := Exec{
			Setup: &ExecSetup{
				Bun: &RuntimeSetup{
					Version: "any",
				},
			},
			Script: `#!/usr/bin/env bun
				import isOdd from 'is-odd'
				console.log(isOdd(3))`,
		}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stdout).To(Equal("true"))
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr", result.Stderr)
	})

	ginkgo.It("should run python with checkout", func() {
		exec := Exec{
			Checkout: &connection.GitConnection{
				URL: "https://github.com/flanksource/artifacts",
			},
			Setup: &ExecSetup{
				Python: &RuntimeSetup{
					Version: "any",
				},
			},
			Script: `#!/usr/bin/env python3
# /// script
# dependencies = [
#   "pyyaml",
# ]
# ///

import yaml

workflow_path = ".github/workflows/lint.yml"

with open(workflow_path, "r", encoding="utf-8") as f:
		data = yaml.safe_load(f)
title = data.get("name")

print(title)`,
		}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stdout).To(Equal("Lint"))
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr", result.Stderr)
	})

	ginkgo.It("should run python3", func() {
		exec := Exec{
			Setup: &ExecSetup{
				Python: &RuntimeSetup{
					Version: "3.10.2",
				},
			},
			Script: `#!/usr/bin/env python3
import platform
print(platform.python_version())
`,
		}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stdout).To(Equal("3.10.2"))
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr", result.Stderr)
	})

	ginkgo.It("should run python3 with packages", func() {
		exec := Exec{
			Setup: &ExecSetup{
				Python: &RuntimeSetup{
					Version: "3.10.2",
				},
			},
			Script: `#!/usr/bin/env python3
# /// script
# dependencies = [
#   "is-even",
# ]
# ///
from is_even import is_even
print(is_even(2))
`,
		}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stdout).To(Equal("True"))
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr", result.Stderr)
	})
})

var _ = ginkgo.Describe("Environment Variables", func() {
	ginkgo.AfterEach(func() {
		os.RemoveAll("./shell-tmp/")
	})

	ctx := context.New()

	ginkgo.It("should access custom env vars", func() {
		exec := Exec{
			Script: "env",
			EnvVars: []types.EnvVar{
				{Name: "mc_test_secret", ValueStatic: "abcdef"},
			},
		}
		expectedVars := []string{"mc_test_secret"}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr")

		envVarKeys := extractEnvVarKeys(result.Stdout)
		expected := append(collections.MapKeys(allowedEnvVars), expectedVars...)
		Expect(lo.Every(expected, envVarKeys)).To(BeTrue(), "expected env vars: %v, got: %v", expectedVars, envVarKeys)
	})

	ginkgo.It("should access multiple custom env vars", func() {
		exec := Exec{
			Script: "env",
			EnvVars: []types.EnvVar{
				{Name: "mc_test_secret_key", ValueStatic: "abc"},
				{Name: "mc_test_secret_id", ValueStatic: "xyz"},
			},
		}
		expectedVars := []string{"mc_test_secret_key", "mc_test_secret_id"}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr")

		envVarKeys := extractEnvVarKeys(result.Stdout)
		expected := append(collections.MapKeys(allowedEnvVars), expectedVars...)
		Expect(lo.Every(expected, envVarKeys)).To(BeTrue(), "expected env vars: %v, got: %v", expectedVars, envVarKeys)
	})

	ginkgo.It("should not access process env", func() {
		exec := Exec{
			Script: "env",
		}

		result, err := Run(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "failed to run command")
		Expect(result.ExitCode).To(Equal(0), "unexpected non-zero exit code")
		Expect(result.Stderr).To(BeEmpty(), "unexpected stderr")

		envVarKeys := extractEnvVarKeys(result.Stdout)
		expected := collections.MapKeys(allowedEnvVars)
		Expect(lo.Every(expected, envVarKeys)).To(BeTrue(), "expected env vars: %v, got: %v", []string{}, envVarKeys)
	})
})

func extractEnvVarKeys(stdout string) []string {
	envVars := strings.Split(stdout, "\n")
	envVars = lo.Filter(envVars, func(v string, _ int) bool {
		key, _, _ := strings.Cut(v, "=")
		return key != "PWD" && key != "SHLVL" && key != "_"
	})
	return lo.Map(envVars, func(v string, _ int) string {
		key, _, _ := strings.Cut(v, "=")
		return key
	})
}

var _ = ginkgo.Describe("PrepareEnvironment", ginkgo.Label("slow"), func() {
	ctx := context.New()

	ginkgo.It("should setup git checkout correctly", func() {
		exec := Exec{
			Checkout: &connection.GitConnection{
				URL:    "https://github.com/flanksource/artifacts",
				Branch: "main",
			},
		}

		cmdCtx, err := prepareEnvironment(ctx, exec)
		Expect(err).ToNot(HaveOccurred(), "prepareEnvironment failed")

		Expect(cmdCtx.mountPoint).ToNot(BeEmpty(), "expected mountPoint to be set")
		Expect(cmdCtx.mountPoint).To(HavePrefix("exec-checkout/"), "expected mountPoint to be in 'exec-checkout/' directory")

		_, err = os.Stat(cmdCtx.mountPoint)
		Expect(err).ToNot(HaveOccurred(), "mount point directory does not exist: %s", cmdCtx.mountPoint)

		Expect(cmdCtx.extra["git"]).ToNot(BeNil(), "expected 'git' key in extra metadata")

		gitURL, ok := cmdCtx.extra["git"].(string)
		Expect(ok).To(BeTrue(), "expected git URL to be a string")
		Expect(gitURL).To(ContainSubstring("github.com/flanksource/artifacts"), "expected git URL to contain 'github.com/flanksource/artifacts'")

		Expect(cmdCtx.extra["commit"]).ToNot(BeNil(), "expected 'commit' key in extra metadata")

		commitHash, ok := cmdCtx.extra["commit"].(string)
		Expect(ok).To(BeTrue(), "expected commit hash to be a string")
		Expect(commitHash).ToNot(BeEmpty(), "expected non-empty commit hash")

		goModPath := filepath.Join(cmdCtx.mountPoint, "go.mod")
		_, err = os.Stat(goModPath)
		Expect(err).ToNot(HaveOccurred(), "expected go.mod to exist in mount point at %s", goModPath)
	})
})

var _ = ginkgo.Describe("DetectInterpreterFromShebang", func() {
	ginkgo.DescribeTable("interpreter detection",
		func(script, expectedInterpreter string, expectedArgs []string) {
			interpreter, args := DetectInterpreterFromShebang(script)
			Expect(interpreter).To(Equal(expectedInterpreter))
			Expect(args).To(Equal(expectedArgs))
		},
		ginkgo.Entry("python via env", "#!/usr/bin/env python\nprint('hello')", "python", []string{"-c"}),
		ginkgo.Entry("python3 with arg", "#!/usr/bin/python3 -u\nprint('hello')", "/usr/bin/python3", []string{"-u", "-c"}),
		ginkgo.Entry("node via env", "#!/usr/bin/env node\nconsole.log('hello')", "node", []string{"-e"}),
		ginkgo.Entry("other default flag", "#!/bin/sh\necho hello", "/bin/sh", []string{"-c"}),
		ginkgo.Entry("pwsh via env", "#!/usr/bin/env pwsh\nWrite-Host 'hello'", "pwsh", []string{}),
		ginkgo.Entry("powershell via env", "#!/usr/bin/env powershell\nWrite-Host 'hello'", "powershell", []string{}),
	)
})
