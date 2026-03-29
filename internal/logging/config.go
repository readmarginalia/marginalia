package logging

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	otlploghttp "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func CreateLogger(ctx context.Context) (*slog.Logger, func(context.Context) error, error) {
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("marginalia"),
			semconv.ServiceVersion("0.0.1"),
			semconv.DeploymentEnvironmentName("local"),
		),
	)

	if err != nil {
		return nil, nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	logger := slog.New(otelslog.NewHandler("app",
		otelslog.WithLoggerProvider(provider),
	))

	shutdown := func(ctx context.Context) error {
		return provider.Shutdown(ctx)
	}

	return logger, shutdown, nil
}
