package service

import (
	"context"
	"fmt"

	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

// Client a setup of connection
type Client struct {
	server   *Service
	coapConn *coapTcpClient.Conn
	handler  ServiceHandler

	deviceID string
}

// newClient creates and initializes client
func newClient(server *Service, client *coapTcpClient.Conn, handler ServiceHandler) *Client {
	return &Client{
		server:   server,
		coapConn: client,
		handler:  handler,
	}
}

func (c *Client) GetCoapConnection() *coapTcpClient.Conn {
	return c.coapConn
}

func (c *Client) GetServiceHandler() ServiceHandler {
	return c.handler
}

func (c *Client) GetDeviceID() string {
	return c.deviceID
}

func (c *Client) SetDeviceID(deviceID string) {
	c.deviceID = deviceID
}

func (c *Client) RemoteAddrString() string {
	return c.coapConn.RemoteAddr().String()
}

func (c *Client) Context() context.Context {
	return c.coapConn.Context()
}

// Close closes coap connection
func (c *Client) Close() error {
	if err := c.coapConn.Close(); err != nil {
		return fmt.Errorf("cannot close client: %w", err)
	}
	return nil
}

// OnClose is invoked when the coap connection was closed.
func (c *Client) OnClose() {
	log.Debugf("close client %v", c.coapConn.RemoteAddr())
}
