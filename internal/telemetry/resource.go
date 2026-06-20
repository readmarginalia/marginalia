package telemetry

import (
	"context"
	"marginalia/internal/buildinfo"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func BuildResource(ctx context.Context, env string) (*resource.Resource, error) {
	return resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName("marginalia"),
			semconv.ServiceVersion(buildinfo.Version),
			semconv.DeploymentEnvironmentName(env),
		),
	)
}
