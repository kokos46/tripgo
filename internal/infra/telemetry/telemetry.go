package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Providers struct {
	Tracer   trace.Tracer
	Meter    metric.Meter
	Shutdown func(context.Context) error
}

func Setup(ctx context.Context) (Providers, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))}))
	slog.SetDefault(logger)
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return Providers{Tracer: otel.Tracer("tripgo"), Meter: otel.Meter("tripgo"), Shutdown: func(context.Context) error { return nil }}, nil
	}
	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint+"/v1/traces"))
	if err != nil {
		return Providers{}, err
	}
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(endpoint+"/v1/metrics"))
	if err != nil {
		return Providers{}, err
	}
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter))
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	return Providers{Tracer: tracerProvider.Tracer("tripgo"), Meter: meterProvider.Meter("tripgo"), Shutdown: func(shutdownCtx context.Context) error {
		if err := meterProvider.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return tracerProvider.Shutdown(shutdownCtx)
	}}, nil
}
func parseLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
