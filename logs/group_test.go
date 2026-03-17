package logs

import (
	"os"
	"testing"

	"github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

type groupCase struct {
	Name     string             `yaml:"name"`
	Config   FieldMappingConfig `yaml:"config"`
	Input    LogResult          `yaml:"input"`
	Expected LogResult          `yaml:"expected"`
}

type groupFixture struct {
	Cases []groupCase `yaml:"cases"`
}

func TestGroupLogs(t *testing.T) {
	data, err := os.ReadFile("testdata/group_logs.yaml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	var fixture groupFixture
	if err := yaml.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			for _, line := range tc.Input.Logs {
				line.SetHash()
			}

			result := &tc.Input
			GroupLogs(result, tc.Config)

			g.Expect(len(result.Logs)).To(gomega.Equal(len(tc.Expected.Logs)), "logs count")
			g.Expect(len(result.Groups)).To(gomega.Equal(len(tc.Expected.Groups)), "groups count")

			for i, expLog := range tc.Expected.Logs {
				assertLogLine(g, result.Logs[i], expLog, "logs[%d]", i)
			}

			for i, expGroup := range tc.Expected.Groups {
				if len(expGroup.Labels) > 0 {
					g.Expect(result.Groups[i].Labels).To(gomega.Equal(expGroup.Labels), "groups[%d].labels", i)
				} else {
					g.Expect(result.Groups[i].Labels).To(gomega.Or(gomega.BeNil(), gomega.BeEmpty()), "groups[%d].labels should be empty", i)
				}
				g.Expect(len(result.Groups[i].Logs)).To(gomega.Equal(len(expGroup.Logs)), "groups[%d].logs count", i)
				for j, expLine := range expGroup.Logs {
					assertLogLine(g, result.Groups[i].Logs[j], expLine, "groups[%d].logs[%d]", i, j)
				}
			}
		})
	}
}

func assertLogLine(g gomega.Gomega, actual *LogLine, expected *LogLine, msgFmt string, args ...any) {
	g.Expect(actual.Message).To(gomega.Equal(expected.Message), append([]any{msgFmt + ".message"}, args...)...)
	if expected.Count > 0 {
		g.Expect(actual.Count).To(gomega.Equal(expected.Count), append([]any{msgFmt + ".count"}, args...)...)
	}
	if expected.Severity != "" {
		g.Expect(actual.Severity).To(gomega.Equal(expected.Severity), append([]any{msgFmt + ".severity"}, args...)...)
	}
	if len(expected.Labels) > 0 {
		g.Expect(actual.Labels).To(gomega.Equal(expected.Labels), append([]any{msgFmt + ".labels"}, args...)...)
	} else if expected.Labels == nil {
		g.Expect(actual.Labels).To(gomega.Or(gomega.BeNil(), gomega.BeEmpty()), append([]any{msgFmt + ".labels should be empty"}, args...)...)
	}
}
