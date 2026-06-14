package telemetry

import (
	"context"
	"os"

	"marginalia/internal/buildinfo"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func BuildResource(ctx context.Context) (*resource.Resource, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

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
