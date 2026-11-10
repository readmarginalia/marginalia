package recommendations

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var tracer = otel.Tracer("marginalia.recommendations")

var meter = otel.Meter("marginalia.recommendations")

var recommendationsCounter, _ = meter.Int64Counter(
	"recommendations.count",
	metric.WithDescription("Counts the number of recommendations added"),
)

func recordRecommendationAdded(ctx context.Context) {
	recommendationsCounter.Add(ctx, 1)
}
