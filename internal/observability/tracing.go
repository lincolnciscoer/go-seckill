package observability

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go-seckill/internal/config"
)

func SetupTracing(
	ctx context.Context,
	serviceName string,
	cfg config.ObservabilityConfig,
	logger *zap.Logger,
) (func(context.Context) error, error) {
	if !cfg.TraceEnabled || cfg.TraceEndpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.TraceEndpoint),
	}
	if cfg.TraceInsecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		logger.Warn("trace exporter init failed, fallback to no-op tracing", zap.Error(err))
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("deployment.environment", "dev"),
		),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.TraceSampleRatio)),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}

func GinTraceMiddleware(serviceName string, cfg config.ObservabilityConfig) gin.HandlerFunc {
	if !cfg.TraceEnabled {
		return func(c *gin.Context) { c.Next() }
	}

	return otelgin.Middleware(serviceName)
}

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
