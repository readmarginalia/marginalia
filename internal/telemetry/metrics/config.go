package metrics

import (
	"context"
	"marginalia/internal/configuration"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func SetupMetrics(ctx context.Context, res *resource.Resource, config configuration.AppConfig) (func(context.Context) error, error) {

	shutdown := func(context.Context) error { return nil }

	if config.OtelEndpoint == "" {
		return shutdown, nil
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(config.OtelEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	reader := metric.NewPeriodicReader(
		exporter,
		metric.WithInterval(15*time.Second),
		metric.WithProducer(runtime.NewProducer()),
	)

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)

	otel.SetMeterProvider(provider)

	if err := runtime.Start(
		runtime.WithMeterProvider(provider),
	); err != nil {
		_ = provider.Shutdown(ctx)
		return nil, err
	}

	return provider.Shutdown, nil
}
