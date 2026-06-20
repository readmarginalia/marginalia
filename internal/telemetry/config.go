package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"marginalia/internal/configuration"
	"marginalia/internal/telemetry/logging"
	"marginalia/internal/telemetry/metrics"
	"marginalia/internal/telemetry/tracing"
)

func Init(ctx context.Context, config configuration.AppConfig) (*slog.Logger, func(context.Context) error, error) {
	res, err := BuildResource(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	logger, shutdownLogs, err := logging.CreateLogger(ctx, res, config)
	if err != nil {
		return nil, nil, err
	}

	shutdownTracing, err := tracing.SetupTracing(ctx, res, config)
	if err != nil {
		_ = shutdownLogs(ctx)
		return nil, nil, err
	}

	shutdownMetrics, err := metrics.SetupMetrics(ctx, res, config)
	if err != nil {
		_ = shutdownLogs(ctx)
		_ = shutdownTracing(ctx)
		return nil, nil, err
	}

	shutdown := func(ctx context.Context) error {
		return errors.Join(
			shutdownMetrics(ctx),
			shutdownTracing(ctx),
			shutdownLogs(ctx),
		)
	}

	return logger, shutdown, nil
}
