package migrate

import (
	"crypto/sha1"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsMarkedForAlwaysRun(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "header comment present",
			content: `-- some comment
-- runs: always
-- another comment
SELECT * FROM table;`,
			expected: true,
		},
		{
			name: "header comment with empty lines",
			content: `
-- some comment

-- runs: always

SELECT * FROM table;`,
			expected: true,
		},
		{
			name: "no header comment",
			content: `-- some comment
-- another comment
SELECT * FROM table;`,
			expected: false,
		},
		{
			name: "comment appears after code",
			content: `-- some comment
SELECT * FROM table;
-- runs: always`,
			expected: false,
		},
		{
			name:     "only header comment",
			content:  "-- runs: always",
			expected: true,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
		{
			name:     "only whitespace",
			content:  "   \n  \t  \n   ",
			expected: false,
		},
		{
			name: "no space after colon",
			content: `-- runs:always
SELECT * FROM table;`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMarkedForAlwaysRun(tt.content); got != tt.expected {
				t.Errorf("isMarkedForAlwaysRun() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "multiple directives",
			content: `-- runs: always
-- if: 'config_access_logs' in tables
-- dependsOn: functions/generate_ulid.sql
DO $$ BEGIN END $$;`,
			expected: map[string]string{
				"runs":      "always",
				"if":        "'config_access_logs' in tables",
				"dependsOn": "functions/generate_ulid.sql",
			},
		},
		{
			name:     "no directives",
			content:  "SELECT 1;",
			expected: map[string]string{},
		},
		{
			name: "stops at non-comment line",
			content: `-- runs: always
SELECT 1;
-- if: should_not_parse`,
			expected: map[string]string{"runs": "always"},
		},
		{
			name:     "trims whitespace in values",
			content:  `-- if:   'my_table' in tables  `,
			expected: map[string]string{"if": "'my_table' in tables"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHeader(tt.content)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("parseHeader() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	t.Run("no directives uses plain content hash", func(t *testing.T) {
		content := "SELECT 1;"
		got := computeHash(content, nil)
		want := sha1.Sum([]byte(content))
		if got != want {
			t.Errorf("computeHash without directives should equal sha1(content)")
		}
	})

	t.Run("content change produces different hash", func(t *testing.T) {
		content1 := "-- if: true\nSELECT 1;"
		content2 := "-- if: true\nSELECT 2;"
		if computeHash(content1, nil) == computeHash(content2, nil) {
			t.Errorf("different content should produce different hashes")
		}
	})

	t.Run("if directive changes hash when CEL result changes", func(t *testing.T) {
		content := "-- if: 'my_table' in tables\nSELECT 1;"

		envWith := map[string]any{"tables": []string{"my_table"}, "properties": map[string]string{}}
		envWithout := map[string]any{"tables": []string{"other_table"}, "properties": map[string]string{}}

		hashWith := computeHash(content, envWith)
		hashWithout := computeHash(content, envWithout)

		if hashWith == hashWithout {
			t.Errorf("hash should differ when if expression result changes")
		}
	})

	t.Run("if directive with properties", func(t *testing.T) {
		content := "-- if: properties['feature'] == 'true'\nSELECT 1;"

		envOn := map[string]any{"tables": []string{}, "properties": map[string]string{"feature": "true"}}
		envOff := map[string]any{"tables": []string{}, "properties": map[string]string{"feature": "false"}}

		hashOn := computeHash(content, envOn)
		hashOff := computeHash(content, envOff)

		if hashOn == hashOff {
			t.Errorf("hash should differ when property value changes")
		}
	})
}
