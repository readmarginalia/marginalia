package logging

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	otlploghttp "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func CreateLogger(ctx context.Context, res *resource.Resource) (*slog.Logger, func(context.Context) error, error) {

	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handlers := []slog.Handler{stdoutHandler}
	shutdown := func(context.Context) error { return nil }

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	if otelEndpoint != "" {
		exporter, err := otlploghttp.New(ctx,
			otlploghttp.WithEndpoint(otelEndpoint),
			otlploghttp.WithInsecure(),
		)
		if err != nil {
			return nil, nil, err
		}

		provider := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		)

		otelHandler := otelslog.NewHandler("marginalia",
			otelslog.WithLoggerProvider(provider),
		)

		handlers = append(handlers, otelHandler)
		shutdown = provider.Shutdown
	}

	logger := slog.New(
		CreateContextHandler(
			CreateMultiHandler(handlers...),
		),
	)

	return logger, shutdown, nil
}
