package telemetry

import (
	"os"

	"marginalia/internal/buildinfo"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func BuildResource() (*resource.Resource, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("marginalia"),
			semconv.ServiceVersion(buildinfo.Version),
			semconv.DeploymentEnvironmentName(env),
		),
	)
}
