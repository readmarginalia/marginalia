package logging

import (
	"context"
	"log/slog"
	"marginalia/internal/correlation"
	"marginalia/internal/requestmeta"

	"go.opentelemetry.io/otel/trace"
)

type contextHandler struct {
	handler slog.Handler
}

func CreateContextHandler(handler slog.Handler) *contextHandler {
	return &contextHandler{handler: handler}
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	record := r.Clone()

	if correlationId, ok := correlation.FromContext(ctx); ok {
		record.AddAttrs(slog.String("correlation_id", correlationId))
	}

	if requestMeta, ok := requestmeta.FromContext(ctx); ok {
		record.AddAttrs(
			slog.String("method", requestMeta.Method),
			slog.String("path", requestMeta.Path),
			slog.String("remote_addr", requestMeta.RemoteAddr),
		)
	}

	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	return h.handler.Handle(ctx, record)
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{handler: h.handler.WithGroup(name)}
}
