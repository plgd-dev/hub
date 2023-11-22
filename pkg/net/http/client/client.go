package client

import (
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Server handles gRPC requests to the service.
type Client struct {
	client    *http.Client
	closeFunc fn.FuncList
}

func (c *Client) HTTP() *http.Client {
	return c.client
}

func (c *Client) AddCloseFunc(f func()) {
	c.closeFunc.AddFunc(f)
}

func (c *Client) Close() {
	c.client.CloseIdleConnections()
	c.closeFunc.Execute()
}

func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Client, error) {
	certManager, err := client.New(config.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = config.MaxIdleConns
	t.MaxConnsPerHost = config.MaxConnsPerHost
	t.MaxIdleConnsPerHost = config.MaxIdleConnsPerHost
	t.IdleConnTimeout = config.IdleConnTimeout
	t.TLSClientConfig = certManager.GetTLSConfig()
	c := &Client{
		client: &http.Client{
			Transport: otelhttp.NewTransport(t, otelhttp.WithTracerProvider(tracerProvider)),
			Timeout:   config.Timeout,
		},
	}
	c.AddCloseFunc(certManager.Close)
	return c, nil
}
