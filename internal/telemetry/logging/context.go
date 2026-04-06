package logging

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

var contextLoggerKey = loggerKey{}

func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(contextLoggerKey).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}

func WithComponent(ctx context.Context, component string) *slog.Logger {
	logger := FromContext(ctx)
	return logger.With("component", component)
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, logger)
}
