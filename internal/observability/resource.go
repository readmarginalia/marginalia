package observability

import (
	"os"

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
			semconv.ServiceVersion("0.1.0"),
			semconv.DeploymentEnvironmentName(env),
		),
	)
}
