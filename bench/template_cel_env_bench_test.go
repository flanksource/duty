package bench_test

import (
	"fmt"
	"testing"
	"time"

	dutyctx "github.com/flanksource/duty/context"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// BenchmarkRunTemplateCELCacheHitRegisteredEnvFuncs isolates the duty-side
// overhead of constructing registered CEL env functions on every RunTemplate
// call. The expression does not call any registered function; the benchmark only
// varies how many functions are registered globally.
func BenchmarkRunTemplateCELCacheHitRegisteredEnvFuncs(b *testing.B) {
	env := map[string]any{
		"id":          "0192f0a4-1234-7000-8000-aaaaaaaaaaaa",
		"namespace":   "default",
		"name":        "nginx-7c5ddbdf54-abcde",
		"index":       42,
		"config_type": "Kubernetes::Pod",
	}

	for _, registeredFuncs := range []int{0, 18, 64} {
		b.Run(fmt.Sprintf("registered=%02d", registeredFuncs), func(b *testing.B) {
			installBenchmarkCelEnvFuncs(b, registeredFuncs)

			ctx := dutyctx.New()
			tmpl := gomplate.Template{
				Expression: `config_type == "Kubernetes::Pod"`,
				CacheKey:   fmt.Sprintf("benchmark.run-template.cel-env-funcs.%d", registeredFuncs),
				CacheTime:  time.Hour,
			}

			// Warm gomplate's compiled CEL program cache. Any measured allocation after
			// this point should be steady-state RunTemplate overhead, not CEL compile.
			if _, err := ctx.RunTemplateBool(tmpl, env); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ok, err := ctx.RunTemplateBool(tmpl, env)
				if err != nil {
					b.Fatal(err)
				}
				if !ok {
					b.Fatal("expected expression to evaluate true")
				}
			}
		})
	}
}

func installBenchmarkCelEnvFuncs(b *testing.B, count int) {
	b.Helper()

	oldCelEnvFuncs := dutyctx.CelEnvFuncs
	oldTemplateFuncs := dutyctx.TemplateFuncs

	dutyctx.CelEnvFuncs = make(map[string]func(dutyctx.Context) cel.EnvOption, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("bench.func_%02d", i)
		dutyctx.CelEnvFuncs[name] = benchmarkCelEnvFunc(name, fmt.Sprintf("bench_func_%02d_string", i))
	}
	dutyctx.TemplateFuncs = nil

	b.Cleanup(func() {
		dutyctx.CelEnvFuncs = oldCelEnvFuncs
		dutyctx.TemplateFuncs = oldTemplateFuncs
	})
}

func benchmarkCelEnvFunc(name, overloadID string) func(dutyctx.Context) cel.EnvOption {
	return func(ctx dutyctx.Context) cel.EnvOption {
		return cel.Function(name,
			cel.Overload(overloadID,
				[]*cel.Type{cel.StringType},
				cel.StringType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					// Capture ctx like production DB/catalog functions do. This binding is
					// intentionally never called by the benchmark expression.
					_ = ctx
					return celtypes.String(fmt.Sprintf("%s:%v", name, arg.Value()))
				}),
			),
		)
	}
}
