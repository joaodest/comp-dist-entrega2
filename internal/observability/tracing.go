package observability

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func SetupTracing(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	if envName := os.Getenv("OTEL_SERVICE_NAME"); envName != "" {
		serviceName = envName
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	sampleRatio := 1.0
	if raw := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed >= 0 && parsed <= 1 {
			sampleRatio = parsed
		}
	}

	res, err := resource.New(ctx, resource.WithAttributes(attribute.String("service.name", serviceName)))
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}

func HTTPHandler(name string, handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, name)
}

func GRPCServerOption() grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler())
}

func GRPCClientOption() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}
