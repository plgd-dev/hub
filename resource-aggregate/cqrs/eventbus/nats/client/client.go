package client

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	conn      *nats.Conn
	closeFunc fn.FuncList
}

func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tp trace.TracerProvider) (*Client, error) {
	certManager, err := client.New(config.TLS, fileWatcher, logger, tp)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	config.Options = append(config.Options, nats.Secure(certManager.GetTLSConfig()), nats.MaxReconnects(-1), nats.FlusherTimeout(config.FlusherTimeout))

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
	c.closeFunc.AddFunc(f)
}

func (c *Client) Close() {
	c.conn.Close()
	c.closeFunc.Execute()
}
