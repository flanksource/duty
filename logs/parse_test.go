package logs

import (
	"os"
	"testing"

	"github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

type parseExpectation struct {
	Message          string            `yaml:"message,omitempty"`
	EffectiveMessage string            `yaml:"effectiveMessage,omitempty"`
	Severity         string            `yaml:"severity,omitempty"`
	Source           string            `yaml:"source,omitempty"`
	Host             string            `yaml:"host,omitempty"`
	Labels           map[string]string `yaml:"labels,omitempty"`
}

type parseCase struct {
	Name          string             `yaml:"name"`
	Input         string             `yaml:"input"`
	Labels        map[string]string  `yaml:"labels,omitempty"`
	MessageFields []string           `yaml:"messageFields,omitempty"`
	Expectations  []parseExpectation `yaml:"expectations"`
}

type parseFixture struct {
	Cases []parseCase `yaml:"cases"`
}

func loadParseFixture(t *testing.T, path string) parseFixture {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", path, err)
	}
	var f parseFixture
	if err := yaml.Unmarshal(data, &f); err != nil {
		t.Fatalf("parsing fixture %s: %v", path, err)
	}
	return f
}

func runParseFixture(t *testing.T, path string, parseFn func(*LogLine)) {
	t.Helper()
	fixture := loadParseFixture(t, path)
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			line := &LogLine{Message: tc.Input, Labels: tc.Labels, Count: 1}
			parseFn(line)
			for _, exp := range tc.Expectations {
				g.Expect(line.Message).To(gomega.Equal(exp.Message), "message")
				g.Expect(line.Severity).To(gomega.Equal(exp.Severity), "severity")
				g.Expect(line.Source).To(gomega.Equal(exp.Source), "source")
				g.Expect(line.Host).To(gomega.Equal(exp.Host), "host")
				for k, v := range exp.Labels {
					g.Expect(line.Labels).To(gomega.HaveKeyWithValue(k, v))
				}
			}
		})
	}
}

func TestParseKlogfmt(t *testing.T) {
	runParseFixture(t, "testdata/parse_klogfmt.yaml", ParseKlogfmt)
}

func TestParseLogfmt(t *testing.T) {
	runParseFixture(t, "testdata/parse_logfmt.yaml", ParseLogfmt)
}

func TestParseJSON(t *testing.T) {
	runParseFixture(t, "testdata/parse_json.yaml", ParseJSON)
}

func TestParseSyslog(t *testing.T) {
	runParseFixture(t, "testdata/parse_syslog.yaml", ParseSyslog)
}

func TestParseAutodetect(t *testing.T) {
	runParseFixture(t, "testdata/parse_autodetect.yaml", ParseAutodetect)
}

func TestEffectiveMessage(t *testing.T) {
	fixture := loadParseFixture(t, "testdata/effective_message.yaml")
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			line := LogLine{Message: tc.Input, Labels: tc.Labels}
			for _, exp := range tc.Expectations {
				g.Expect(line.EffectiveMessage(tc.MessageFields...)).To(gomega.Equal(exp.EffectiveMessage))
			}
		})
	}
}
