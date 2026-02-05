package migrate

import "testing"

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
			name: "similar but incorrect comment",
			content: `-- runs:always
SELECT * FROM table;`,
			expected: false,
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
