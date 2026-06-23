package context

import (
	"sync/atomic"
	"testing"

	"github.com/flanksource/gomplate/v3"
	"github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	. "github.com/onsi/gomega"
)

func TestRunTemplateOnlyBuildsReferencedCelEnvFuncs(t *testing.T) {
	g := NewWithT(t)
	oldCelEnvFuncs := CelEnvFuncs
	oldTemplateFuncs := TemplateFuncs
	defer func() {
		CelEnvFuncs = oldCelEnvFuncs
		TemplateFuncs = oldTemplateFuncs
	}()

	var usedBuilt atomic.Int32
	var unusedBuilt atomic.Int32
	CelEnvFuncs = map[string]func(Context) cel.EnvOption{
		"bench.usedCel": func(ctx Context) cel.EnvOption {
			usedBuilt.Add(1)
			return cel.Function("bench.used",
				cel.Overload("bench_used_string",
					[]*cel.Type{cel.StringType},
					cel.BoolType,
					cel.UnaryBinding(func(arg ref.Val) ref.Val {
						return celtypes.Bool(arg.Value() == "default")
					}),
				),
			)
		},
		"bench.unused": func(ctx Context) cel.EnvOption {
			unusedBuilt.Add(1)
			return cel.Function("bench.unused",
				cel.Overload("bench_unused_string",
					[]*cel.Type{cel.StringType},
					cel.StringType,
					cel.UnaryBinding(func(arg ref.Val) ref.Val {
						return celtypes.String(arg.Value().(string))
					}),
				),
			)
		},
	}
	TemplateFuncs = nil

	ctx := New()
	env := map[string]any{"config_namespace": "default"}

	ok, err := ctx.RunTemplateBool(gomplate.Template{
		Expression: `config_namespace == "default"`,
		CacheKey:   "test.run-template.no-registered-cel-func",
	}, env)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ok).To(BeTrue())
	g.Expect(usedBuilt.Load()).To(Equal(int32(0)))
	g.Expect(unusedBuilt.Load()).To(Equal(int32(0)))

	ok, err = ctx.RunTemplateBool(gomplate.Template{
		Expression: `bench.used(config_namespace)`,
		CacheKey:   "test.run-template.referenced-cel-func",
	}, env)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ok).To(BeTrue())
	g.Expect(usedBuilt.Load()).To(Equal(int32(1)))
	g.Expect(unusedBuilt.Load()).To(Equal(int32(0)))
}

func TestCelExpressionCalls(t *testing.T) {
	tests := []struct {
		name string
		expr string
		fn   string
		want bool
	}{
		{name: "direct namespaced call", expr: `bench.used(config_namespace)`, fn: "bench.used", want: true},
		{name: "whitespace before args", expr: `bench.used (config_namespace)`, fn: "bench.used", want: true},
		{name: "prefix is not a call", expr: `bench.used_extra(config_namespace)`, fn: "bench.used", want: false},
		{name: "suffix is not a call", expr: `other.bench.used(config_namespace)`, fn: "bench.used", want: false},
		{name: "similar db function prefix", expr: `db.external_users_all(scraper_id)`, fn: "db.external_users", want: false},
		{name: "exact db function", expr: `db.external_users_all(scraper_id)`, fn: "db.external_users_all", want: true},
		{name: "no function call", expr: `config_type == "Kubernetes::Pod"`, fn: "db.external_users", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(celExpressionCalls(tt.expr, tt.fn)).To(Equal(tt.want))
		})
	}
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
