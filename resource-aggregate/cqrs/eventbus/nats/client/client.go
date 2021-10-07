package client

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/pkg/security/certManager/client"
)

type Client struct {
	conn      *nats.Conn
	closeFunc []func()
}

func New(config Config, logger log.Logger) (*Client, error) {
	certManager, err := client.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	config.Options = append(config.Options, nats.Secure(certManager.GetTLSConfig()), nats.MaxReconnects(-1))

	conn, err := nats.Connect(config.URL, config.Options...)
	if err != nil {
		certManager.Close()
		return nil, fmt.Errorf("cannot create nats client connection: %w", err)
	}
	c := &Client{conn: conn}
	c.AddCloseFunc(certManager.Close)
	return c, nil
}

func (c *Client) GetConn() *nats.Conn {
	return c.conn
}

func (c *Client) AddCloseFunc(f func()) {
	c.closeFunc = append(c.closeFunc, f)
}

func (c *Client) Close() {
	c.conn.Close()
	for _, f := range c.closeFunc {
		f()
	}
}
