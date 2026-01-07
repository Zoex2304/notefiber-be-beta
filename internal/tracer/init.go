package tracer

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracer initializes OpenTelemetry with OTLP HTTP exporter (compatible with Jaeger).
// Returns a shutdown function that should be called on application exit.
func InitTracer() func(context.Context) error {
	ctx := context.Background()

	// Create OTLP HTTP exporter (Jaeger accepts OTLP on port 4318)
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
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
	log.Println("✅ OpenTelemetry tracer initialized with OTLP exporter (Jaeger-compatible)")

	return tp.Shutdown
}
