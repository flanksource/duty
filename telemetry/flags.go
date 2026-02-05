package telemetry

import (
	"os"

	"github.com/spf13/pflag"
)

func BindFlags(flags *pflag.FlagSet, serviceName string) {
	flags.StringVar(&OtelCollectorURL, "otel-collector-url", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "OpenTelemetry gRPC Collector URL in host:port format")
	flags.StringVar(&OtelServiceName, "otel-service-name", serviceName, "OpenTelemetry service name for the resource")
	flags.BoolVar(&OtelInsecure, "otel-insecure", true, "Set to true to disable TLS for insecure OpenTelemetry collector")
}
