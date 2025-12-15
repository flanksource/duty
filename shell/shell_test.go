package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flanksource/commons/collections"
	"github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

func TestEnv(t *testing.T) {
	testData := []struct {
		name         string
		exec         Exec
		expectedVars []string
	}{
		{
			name: "access custom env vars",
			exec: Exec{
				Script: "env",
				EnvVars: []types.EnvVar{
					{Name: "mc_test_secret", ValueStatic: "abcdef"},
				},
			},
			expectedVars: []string{"mc_test_secret"},
		},
		{
			name: "access multiple custom env vars",
			exec: Exec{
				Script: "env",
				EnvVars: []types.EnvVar{
					{Name: "mc_test_secret_key", ValueStatic: "abc"},
					{Name: "mc_test_secret_id", ValueStatic: "xyz"},
				},
			},
			expectedVars: []string{"mc_test_secret_key", "mc_test_secret_id"},
		},
		{
			name: "no access to process env",
			exec: Exec{
				Script: "env",
			},
			expectedVars: []string{},
		},
	}

	ctx := context.New()
	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			result, err := Run(ctx, td.exec)
			g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to run command")

			g.Expect(result.ExitCode).To(gomega.Equal(0), "unexpected non-zero exit code")
			g.Expect(result.Stderr).To(gomega.BeEmpty(), "unexpected stderr")

			envVars := strings.Split(result.Stdout, "\n")

			// These env vars are always made available.
			envVars = lo.Filter(envVars, func(v string, _ int) bool {
				key, _, _ := strings.Cut(v, "=")
				return key != "PWD" && key != "SHLVL" && key != "_"
			})

			envVarKeys := lo.Map(envVars, func(v string, _ int) string {
				key, _, _ := strings.Cut(v, "=")
				return key
			})

			expected := collections.MapKeys(allowedEnvVars)
			expected = append(expected, td.expectedVars...)
			g.Expect(lo.Every(expected, envVarKeys)).To(gomega.BeTrue(), "expected env vars: %v, got: %v", td.expectedVars, envVarKeys)
		})

		os.RemoveAll("./shell-tmp/")
	}
}

func TestPrepareEnvironment(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.New()

	exec := Exec{
		Checkout: &connection.GitConnection{
			URL:    "https://github.com/flanksource/artifacts",
			Branch: "main",
		},
	}

	cmdCtx, err := prepareEnvironment(ctx, exec)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "prepareEnvironment failed")

	g.Expect(cmdCtx.mountPoint).ToNot(gomega.BeEmpty(), "expected mountPoint to be set")
	g.Expect(cmdCtx.mountPoint).To(gomega.HavePrefix("exec-checkout/"), "expected mountPoint to be in 'exec-checkout/' directory")

	_, err = os.Stat(cmdCtx.mountPoint)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "mount point directory does not exist: %s", cmdCtx.mountPoint)

	g.Expect(cmdCtx.extra["git"]).ToNot(gomega.BeNil(), "expected 'git' key in extra metadata")

	gitURL, ok := cmdCtx.extra["git"].(string)
	g.Expect(ok).To(gomega.BeTrue(), "expected git URL to be a string")
	g.Expect(gitURL).To(gomega.ContainSubstring("github.com/flanksource/artifacts"), "expected git URL to contain 'github.com/flanksource/artifacts'")

	g.Expect(cmdCtx.extra["commit"]).ToNot(gomega.BeNil(), "expected 'commit' key in extra metadata")

	commitHash, ok := cmdCtx.extra["commit"].(string)
	g.Expect(ok).To(gomega.BeTrue(), "expected commit hash to be a string")
	g.Expect(commitHash).ToNot(gomega.BeEmpty(), "expected non-empty commit hash")

	goModPath := filepath.Join(cmdCtx.mountPoint, "go.mod")
	_, err = os.Stat(goModPath)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "expected go.mod to exist in mount point at %s", goModPath)
}

func TestDetectInterpreterFromShebang(t *testing.T) {
	g := gomega.NewWithT(t)

	testCases := []struct {
		name        string
		script      string
		interpreter string
		args        []string
	}{
		{
			name:        "python via env",
			script:      "#!/usr/bin/env python\nprint('hello')",
			interpreter: "python",
			args:        []string{"-c"},
		},
		{
			name:        "python3 with arg",
			script:      "#!/usr/bin/python3 -u\nprint('hello')",
			interpreter: "/usr/bin/python3",
			args:        []string{"-u", "-c"},
		},
		{
			name:        "node via env",
			script:      "#!/usr/bin/env node\nconsole.log('hello')",
			interpreter: "node",
			args:        []string{"-e"},
		},
		{
			name:        "other default flag",
			script:      "#!/bin/sh\necho hello",
			interpreter: "/bin/sh",
			args:        []string{"-c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interpreter, args := DetectInterpreterFromShebang(tc.script)

			g.Expect(interpreter).To(gomega.Equal(tc.interpreter))
			g.Expect(args).To(gomega.Equal(tc.args))
		})
	}
}
