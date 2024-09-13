package shell

import (
	"strings"
	"testing"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
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
			expectedVars: []string{"mc_test_secret=abcdef"},
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
			expectedVars: []string{"mc_test_secret_key=abc", "mc_test_secret_id=xyz"},
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
			result, err := Run(ctx, td.exec)
			if err != nil {
				t.Fatalf("failed to run command %s", err)
			}

			if result.ExitCode != 0 {
				t.Errorf("unexpected non-zero exit code: %d", result.ExitCode)
			}

			if result.Stderr != "" {
				t.Errorf("unexpected stderr: %s", result.Stderr)
			}

			envVars := strings.Split(result.Stdout, "\n")

			// These env vars are always made available.
			envVars = lo.Filter(envVars, func(v string, _ int) bool {
				key, _, _ := strings.Cut(v, "=")
				return key != "PWD" && key != "SHLVL" && key != "_"
			})

			if !lo.Every(envVars, td.expectedVars) || !lo.Every(td.expectedVars, envVars) {
				t.Errorf("expected %s, got %s", td.expectedVars, envVars)
			}
		})
	}
}
