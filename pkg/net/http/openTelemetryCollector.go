package http

import (
	"net/http"

	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type OpenTelemetryCollectorConfig struct {
	otelClient.Config `yaml:",inline"`
}

func (c *OpenTelemetryCollectorConfig) Validate() error {
	return c.Config.Validate()
}

func OpenTelemetryNewHandler(handler http.Handler, serviceName string, tracerProvider trace.TracerProvider) http.Handler {
	opts := []otelhttp.Option{
		otelhttp.WithTracerProvider(tracerProvider),
	}

	return otelhttp.NewHandler(handler, serviceName, opts...)
}
