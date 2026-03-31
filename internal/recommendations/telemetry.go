package recommendations

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("marginalia/recommendations")
