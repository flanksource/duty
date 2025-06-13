package tracing

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/credentials"

	"github.com/flanksource/commons/logger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Tracer struct {
	ServiceName  string
	CollectorURL string
	Insecure     bool
	Samplers     map[string]sdktrace.Sampler
}

func (tracer Tracer) Sample(name string, perc float64) Tracer {
	tracer.Samplers[name] = NewCounterSampler(perc)
	return tracer
}

func (tracer Tracer) Init() func() {
	var client otlptrace.Client
	if strings.HasPrefix(tracer.CollectorURL, "http") {
		client = otlptracehttp.NewClient(
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithEndpoint(strings.ReplaceAll(tracer.CollectorURL, "https://", "")),
			otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
			otlptracehttp.WithTLSClientConfig(&tls.Config{}))
	} else {
		var secureOption otlptracegrpc.Option

		if !tracer.Insecure {
			secureOption = otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
		} else {
			secureOption = otlptracegrpc.WithInsecure()
		}

		client = otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(tracer.CollectorURL),
		)
	}

	exporter, err := otlptrace.New(
		context.Background(),
		client,
	)

	if err != nil {
		logger.Errorf("Failed to create opentelemetry exporter: %v", err)
		return func() {}
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", tracer.ServiceName),
		),
	)
	if err != nil {
		logger.Errorf("Could not set opentelemetry resources: %v", err)
		return func() {}
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(NewCustomSampler(sdktrace.AlwaysSample(), tracer.Samplers)),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)

	// Register the TraceContext propagator globally.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		err := exporter.Shutdown(ctx)
		if err != nil {
			logger.Errorf(err.Error())
		}
		defer cancel()
	}
}
