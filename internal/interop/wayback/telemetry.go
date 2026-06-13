package wayback

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("marginalia.interop.wayback")

func beginSaveSpan(ctx context.Context, url string) (context.Context, func()) {
	ctx, span := tracer.Start(ctx, "wayback.RequestSave")
	span.SetAttributes(
		attribute.String("url", url),
	)
	return ctx, func() {
		span.End()
	}
}
