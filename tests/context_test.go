package tests

import (
	gocontext "context"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flanksource/duty/context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var exporter *stdouttrace.Exporter
var tracer trace.Tracer

var _ = Describe("Context", func() {
	var err error
	exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(resource.NewSchemaless()),
	)

	tracer = provider.Tracer("example.com/basic")

	It("should record spans", func() {
		c := context.NewContext(gocontext.Background()).WithObject(metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: map[string]string{"debug": "true"},
		})
		c.WithTracer(tracer)

		Expect(c.GetObjectMeta().Name).To(Equal("test"))
		Expect(c.IsDebug()).To(BeTrue())
		Expect(c.IsTrace()).To(BeFalse())

		Expect(c.GetName()).To(Equal("test"))
		Expect(c.GetNamespace()).To(Equal("default"))

		ctx, span := c.StartSpan("test")

		Expect(ctx.GetName()).To(Equal("test"))
		Expect(ctx.GetNamespace()).To(Equal("default"))
		inner := ctx.WithObject(metav1.ObjectMeta{
			Name:        "jane",
			Namespace:   "default",
			Annotations: map[string]string{"trace": "true"},
		})
		Expect(inner.GetName()).To(Equal("jane"))
		Expect(inner.GetNamespace()).To(Equal("default"))

		Expect(inner.IsTrace()).To(BeTrue())
		span.End()
	})
})
