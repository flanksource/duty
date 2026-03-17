package schema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseHCLHeader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "if directive",
			content: `// if: 'config_access_logs' in tables
table "config_access_logs" {
  schema = schema.public
}`,
			expected: map[string]string{"if": "'config_access_logs' in tables"},
		},
		{
			name: "multiple directives",
			content: `// if: 'config_access_logs' in tables
// dependsOn: schema/main.hcl
table "config_access_logs" {}`,
			expected: map[string]string{
				"if":         "'config_access_logs' in tables",
				"dependsOn":  "schema/main.hcl",
			},
		},
		{
			name:     "no directives",
			content:  `table "config_items" {}`,
			expected: map[string]string{},
		},
		{
			name: "stops at non-comment line",
			content: `// if: true
table "t" {}
// if: should_not_parse`,
			expected: map[string]string{"if": "true"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHCLHeader([]byte(tt.content))
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("parseHCLHeader() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
