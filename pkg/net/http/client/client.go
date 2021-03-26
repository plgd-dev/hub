package client

import (
	"fmt"
	"net/http"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"go.uber.org/zap"
)

// Server handles gRPC requests to the service.
type Client struct {
	client    *http.Client
	closeFunc []func()
}

func (c *Client) HTTP() *http.Client {
	return c.client
}

func (s *Client) Close() {
	s.client.CloseIdleConnections()
	for _, f := range s.closeFunc {
		f()
	}
}

func New(config Config, logger *zap.Logger) (*Client, error) {
	certManager, err := client.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = config.MaxIdleConns
	t.MaxConnsPerHost = config.MaxConnsPerHost
	t.MaxIdleConnsPerHost = config.MaxIdleConnsPerHost
	t.IdleConnTimeout = config.IdleConnTimeout
	t.TLSClientConfig = certManager.GetTLSConfig()
	return &Client{
		client: &http.Client{
			Transport: t,
			Timeout:   config.Timeout,
		}, closeFunc: []func(){certManager.Close},
	}, nil
}
