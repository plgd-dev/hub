package otelcoap

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	InstrumentationName = "github.com/plgd-dev/hub/coap-gateway/opentelemetry"
)

// config represents the configuration options available for the http.Handler
// and http.Transport types.
type config struct {
	SpanStartOptions []trace.SpanStartOption
	TracerProvider   trace.TracerProvider
}

// Option interface used for setting optional config properties.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// newConfig creates a new config struct and applies opts to it.
func newConfig(opts ...Option) *config {
	c := &config{
		TracerProvider: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return optionFunc(func(c *config) {
		c.SpanStartOptions = append(c.SpanStartOptions, opts...)
	})
}
