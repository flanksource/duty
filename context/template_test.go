package context

import (
	"testing"

	"github.com/flanksource/gomplate/v3"
	"github.com/google/cel-go/cel"
	. "github.com/onsi/gomega"
)

func TestRunTemplateOnlyBuildsMatchingFunctionContexts(t *testing.T) {
	g := NewWithT(t)
	oldCelEnvFuncs := CelEnvFuncs
	oldTemplateFuncs := TemplateFuncs
	defer func() {
		CelEnvFuncs = oldCelEnvFuncs
		TemplateFuncs = oldTemplateFuncs
	}()

	var celBuilds int
	var templateBuilds int
	CelEnvFuncs = map[string]func(Context) cel.EnvOption{
		"unused_cel_context": func(ctx Context) cel.EnvOption {
			celBuilds++
			return cel.Variable("unused_cel_context", cel.AnyType)
		},
	}
	TemplateFuncs = map[string]func(Context) any{
		"unused_template_context": func(ctx Context) any {
			templateBuilds++
			return func() string { return "unused" }
		},
	}

	ctx := New()
	env := map[string]any{"name": "World"}

	got, err := ctx.RunTemplate(gomplate.Template{Template: "Hello {{.name}}!"}, env)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal("Hello World!"))
	g.Expect(celBuilds).To(Equal(0))
	g.Expect(templateBuilds).To(Equal(1))

	ok, err := ctx.RunTemplateBool(gomplate.Template{Expression: `name == "World"`}, env)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ok).To(BeTrue())
	g.Expect(celBuilds).To(Equal(1))
	g.Expect(templateBuilds).To(Equal(1))
}

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
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := New()

			got, err := ctx.RunTemplate(tt.template, tt.env)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
