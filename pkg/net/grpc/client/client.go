package client

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fn"
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
	closeFunc fn.FuncList
}

func (c *Client) GRPC() *grpc.ClientConn {
	return c.client
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (c *Client) AddCloseFunc(f func()) {
	c.closeFunc.AddFunc(f)
}

func (c *Client) Close() error {
	err := c.client.Close()
	c.closeFunc.Execute()
	return err
}

func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...grpc.DialOption) (*Client, error) {
	err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	certManager, err := client.New(config.TLS, fileWatcher, logger, tracerProvider)
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
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(tracerProvider))),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(config.SendMsgSize), grpc.MaxCallRecvMsgSize(config.RecvMsgSize)),
	}
	if len(opts) > 0 {
		v = append(v, opts...)
	}

	conn, err := grpc.NewClient(config.Addr, v...)
	if err != nil {
		return nil, fmt.Errorf("cannot dial: %w", err)
	}

	c := &Client{
		client: conn,
	}
	c.AddCloseFunc(certManager.Close)
	return c, nil
}
