package context

import (
	"testing"

	"github.com/flanksource/gomplate/v3"
	. "github.com/onsi/gomega"
)

func TestRunTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template gomplate.Template
		env      map[string]any
		want     string
	}{
		{
			name: "simple variable substitution",
			template: gomplate.Template{
				Template: "Hello {{.name}}!",
			},
			env: map[string]any{
				"name": "World",
			},
			want: "Hello World!",
		},
		{
			name: "simple variable substitution $ syntax",
			template: gomplate.Template{
				Template: "Hello $(.name)!",
			},
			env: map[string]any{
				"name": "World",
			},
			want: "Hello World!",
		},
		{
			name: "simple variable substitution $ syntax | explicit delimiters",
			template: gomplate.Template{
				Template:   "Hello $(.name)!",
				LeftDelim:  "{{",
				RightDelim: "}}",
			},
			env: map[string]any{
				"name": "World",
			},
			want: "Hello $(.name)!",
		},
		{
			name: "mixed $ and {{}} syntax variables",
			template: gomplate.Template{
				Template: "{{.greeting}} $(.name)!",
			},
			env: map[string]any{
				"greeting": "Hi",
				"name":     "Alice",
			},
			want: "Hi Alice!",
		},
		{
			name: "conditional statement",
			template: gomplate.Template{
				Template: "{{if .isAdmin}}Admin{{else}}User{{end}}",
			},
			env: map[string]any{
				"isAdmin": true,
			},
			want: "Admin",
		},
		{
			name: "conditional statement with $ syntax",
			template: gomplate.Template{
				Template: "$(if .isAdmin)Admin$(else)User$(end)",
			},
			env: map[string]any{
				"isAdmin": true,
			},
			want: "Admin",
		},
		{
			name: "missing variable",
			template: gomplate.Template{
				Template: "Hello {{.missing}}!",
			},
			env:  map[string]any{},
			want: "Hello <no value>!",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := New()

			got, err := ctx.RunTemplate(tt.template, tt.env)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
