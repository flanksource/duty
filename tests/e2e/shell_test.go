package e2e

import (
	"os"
	"testing"

	"github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/shell"
)

func TestShellRun(t *testing.T) {
	testData := []struct {
		name   string
		exec   shell.Exec
		stdout string
	}{
		{
			name:   "bun",
			stdout: "true",
			exec: shell.Exec{
				Setup: &shell.ExecSetup{
					Bun: &shell.RuntimeSetup{
						Version: "latest",
					},
				},
				Script: `#!/usr/bin/env bun
				import isOdd from 'is-odd'
				console.log(isOdd(3))`,
			},
		},
		{
			name:   "python3",
			stdout: "3.10.2",
			exec: shell.Exec{
				Setup: &shell.ExecSetup{
					Python: &shell.RuntimeSetup{
						Version: "3.10.2",
					},
				},
				Script: `#!/usr/bin/env python3
import platform
print(platform.python_version())
`,
			},
		},
		{
			name:   "python3 with pkgs",
			stdout: "True",
			exec: shell.Exec{
				Setup: &shell.ExecSetup{
					Python: &shell.RuntimeSetup{
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
			},
		},
	}

	ctx := context.New()
	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			result, err := shell.Run(ctx, td.exec)
			g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to run command")

			g.Expect(result.ExitCode).To(gomega.Equal(0), "unexpected non-zero exit code")
			g.Expect(result.Stdout).To(gomega.Equal(td.stdout))
			g.Expect(result.Stderr).To(gomega.BeEmpty(), "unexpected stderr", result.Stderr)
		})

		os.RemoveAll("./shell-tmp/")
	}
}
