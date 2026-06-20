package metrics

import (
	"context"
	"marginalia/internal/configuration"
	"time"

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
	)

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)

	otel.SetMeterProvider(provider)

	return provider.Shutdown, nil
}
