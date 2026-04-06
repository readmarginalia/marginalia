package correlation

import (
	"context"

	"github.com/google/uuid"
)

type CorrelationIdKey struct{}

var correlationIdKey = CorrelationIdKey{}

func WithCorrelationId(ctx context.Context, correlationId string) context.Context {
	return context.WithValue(ctx, correlationIdKey, correlationId)
}

func FromContext(ctx context.Context) (string, bool) {
	correlationId, ok := ctx.Value(correlationIdKey).(string)
	return correlationId, ok
}

func EnsureCorrelationId(ctx context.Context) (context.Context, string) {
	correlationId, ok := FromContext(ctx)
	if !ok || correlationId == "" {
		correlationId = createCorrelationId()
		ctx = WithCorrelationId(ctx, correlationId)
	}
	return ctx, correlationId
}

func createCorrelationId() string {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return ""
	}
	return uuid.String()
}
