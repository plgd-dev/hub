package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type Client struct {
	ctx            context.Context
	logger         log.Logger
	client         *client.Client
	tracerProvider *sdktrace.TracerProvider
	closeFunc      fn.FuncList
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (c *Client) AddCloseFunc(f func()) {
	c.closeFunc.AddFunc(f)
}

func (c *Client) GetTracerProvider() trace.TracerProvider {
	if c.tracerProvider == nil {
		return noop.NewTracerProvider()
	}
	return c.tracerProvider
}

func (c *Client) close(ctx context.Context) error {
	var errors *multierror.Error
	if c.tracerProvider != nil {
		err := c.tracerProvider.Shutdown(ctx)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	c.closeFunc.Execute()
	return errors.ErrorOrNil()
}

func (c *Client) Close() {
	if err := c.close(c.ctx); err != nil {
		c.logger.Errorf("cannot close open telemetry collector client: %w", err)
	}
}

// New creates a new tracer provider with grpc exporter when it is enabled.
func New(ctx context.Context, cfg Config, serviceName string, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Client, error) {
	if !cfg.GRPC.Enabled {
		return &Client{
			ctx:    ctx,
			logger: logger,
		}, nil
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
	client, err := client.New(cfg.GRPC.Connection, fileWatcher, logger, noop.NewTracerProvider())
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
		ctx:            ctx,
		logger:         logger,
		tracerProvider: tracerProvider,
		client:         client,
	}, nil
}
