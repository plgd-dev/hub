package client

import (
	"crypto/tls"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/fn"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type CertificateManager interface {
	GetTLSConfig() *tls.Config
	Close()
}

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

func New(config pkgTls.HTTPConfigurer, cm CertificateManager, tracerProvider trace.TracerProvider) (*Client, error) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = config.GetMaxIdleConns()
	t.MaxConnsPerHost = config.GetMaxConnsPerHost()
	t.MaxIdleConnsPerHost = config.GetMaxIdleConnsPerHost()
	t.IdleConnTimeout = config.GetIdleConnTimeout()
	t.TLSClientConfig = cm.GetTLSConfig()
	c := &Client{
		client: &http.Client{
			Transport: otelhttp.NewTransport(t, otelhttp.WithTracerProvider(tracerProvider)),
			Timeout:   config.GetTimeout(),
		},
	}
	c.AddCloseFunc(cm.Close)
	return c, nil
}
