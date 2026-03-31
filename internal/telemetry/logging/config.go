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
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	otelHandler := otelslog.NewHandler("app",
		otelslog.WithLoggerProvider(provider),
	)

	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(
		CreateMultiHandler(otelHandler, stdoutHandler),
	)

	return logger, provider.Shutdown, nil
}
