package grpc

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	client         *client.Client
	tracerProvider *sdktrace.TracerProvider
	closeFunc      fn.FuncList
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (s *Client) AddCloseFunc(f func()) {
	s.closeFunc.AddFunc(f)
}

func (c *Client) GetTracerProvider() trace.TracerProvider {
	if c.tracerProvider == nil {
		return trace.NewNoopTracerProvider()
	}
	return c.tracerProvider
}

func (s *Client) Close(ctx context.Context) error {
	var errors []error
	if s.tracerProvider != nil {
		err := s.tracerProvider.Shutdown(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if s.client != nil {
		err := s.client.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	s.closeFunc.Execute()
	if len(errors) == 1 {
		return errors[0]
	}
	if len(errors) > 1 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

// New creates a new tracer provider with grpc exporter when it is enabled.
func New(ctx context.Context, logger log.Logger, serviceName string, cfg Config) (*Client, error) {
	if !cfg.Enabled {
		return &Client{}, nil
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns
	client, err := client.New(cfg.Connection, logger, trace.NewNoopTracerProvider())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(client.GRPC()))
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to create trace exporterr: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global trace provider
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return &Client{
		tracerProvider: tracerProvider,
		client:         client,
	}, nil
}
