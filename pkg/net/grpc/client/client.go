package client

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// GRPC Client.
type Client struct {
	client    *grpc.ClientConn
	closeFunc []func()
}

func (c *Client) GRPC() *grpc.ClientConn {
	return c.client
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (c *Client) AddCloseFunc(f func()) {
	c.closeFunc = append(c.closeFunc, f)
}

func (c *Client) Close() error {
	err := c.client.Close()
	for _, f := range c.closeFunc {
		f()
	}
	return err
}

func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...grpc.DialOption) (*Client, error) {
	certManager, err := client.New(config.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	v := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(certManager.GetTLSConfig())),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAlive.Time,
			Timeout:             config.KeepAlive.Timeout,
			PermitWithoutStream: config.KeepAlive.PermitWithoutStream,
		}),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(tracerProvider))),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(tracerProvider))),
	}
	if len(opts) > 0 {
		v = append(v, opts...)
	}

	conn, err := grpc.Dial(config.Addr, v...)
	if err != nil {
		return nil, fmt.Errorf("cannot dial: %w", err)
	}

	return &Client{
		client: conn, closeFunc: []func(){certManager.Close},
	}, nil
}
