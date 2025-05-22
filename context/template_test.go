package context

import (
	"testing"

	"github.com/flanksource/gomplate/v3"
	. "github.com/onsi/gomega"
)

func TestRunTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		env      map[string]any
		want     string
	}{
		{
			name:     "simple variable substitution",
			template: "Hello {{.name}}!",
			env: map[string]any{
				"name": "World",
			},
			want: "Hello World!",
		},
		{
			name:     "multiple variables",
			template: "{{.greeting}} {{.name}}!",
			env: map[string]any{
				"greeting": "Hi",
				"name":     "Alice",
			},
			want: "Hi Alice!",
		},
		{
			name:     "nested object access",
			template: "{{.person.name}} is {{.person.age}} years old",
			env: map[string]any{
				"person": map[string]any{
					"name": "Bob",
					"age":  30,
				},
			},
			want: "Bob is 30 years old",
		},
		{
			name:     "arithmetic operations",
			template: "{{.x}} + {{.y}} = {{add .x .y}}",
			env: map[string]any{
				"x": 5,
				"y": 3,
			},
			want: "5 + 3 = 8",
		},
		{
			name:     "conditional statement",
			template: "{{if .isAdmin}}Admin{{else}}User{{end}}",
			env: map[string]any{
				"isAdmin": true,
			},
			want: "Admin",
		},
		{
			name:     "missing variable error",
			template: "Hello {{.missing}}!",
			env:      map[string]any{},
			want:     "Hello <no value>!",
		},
		{
			name:     "zero state - empty string",
			template: "Hello {{.name}}!",
			env: map[string]any{
				"name": "",
			},
			want: "Hello !",
		},
		{
			name:     "zero state - nil value",
			template: "Value: {{.value}}",
			env: map[string]any{
				"value": nil,
			},
			want: "Value: <no value>",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := New()
			tmpl := gomplate.Template{
				Template: tt.template,
			}

			got, err := ctx.RunTemplate(tmpl, tt.env)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
