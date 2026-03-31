package extract

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("marginalia/extract")
