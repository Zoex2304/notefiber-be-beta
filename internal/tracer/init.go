package tracer

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracer initializes OpenTelemetry with OTLP HTTP exporter (compatible with Jaeger).
// Returns a shutdown function that should be called on application exit.
// Tracing is DISABLED by default. Set OTEL_ENABLED=true to enable.
func InitTracer() func(context.Context) error {
	// Check if tracing is enabled via environment variable
	otelEnabled := os.Getenv("OTEL_ENABLED")
	if otelEnabled != "true" {
		log.Println("OpenTelemetry tracing is disabled (set OTEL_ENABLED=true to enable)")
		return func(context.Context) error { return nil }
	}

	ctx := context.Background()

	// Get OTLP endpoint from environment, default to localhost:4318
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "localhost:4318"
	}

	// Create OTLP HTTP exporter (Jaeger accepts OTLP on port 4318)
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otelEndpoint),
		otlptracehttp.WithInsecure(), // Use HTTP, not HTTPS for local Jaeger
	)
	if err != nil {
		log.Printf("Warning: Failed to create OTLP exporter: %v (tracing disabled)", err)
		return func(context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("ai-notetaking-backend"),
		)),
	)

	otel.SetTracerProvider(tp)
	log.Printf("✅ OpenTelemetry tracer initialized (endpoint: %s)", otelEndpoint)

	return tp.Shutdown
}
