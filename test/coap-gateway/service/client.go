package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

// Client a setup of connection
type Client struct {
	server   *Service
	coapConn *tcp.ClientConn
	handler  ServiceHandler

	deviceID string
}

// newClient creates and initializes client
func newClient(server *Service, client *tcp.ClientConn, handler ServiceHandler) *Client {
	return &Client{
		server:   server,
		coapConn: client,
		handler:  handler,
	}
}

func (client *Client) GetCoapConnection() *tcp.ClientConn {
	return client.coapConn
}

func (client *Client) GetServiceHandler() ServiceHandler {
	return client.handler
}

func (client *Client) GetDeviceID() string {
	return client.deviceID
}

func (client *Client) SetDeviceID(deviceID string) {
	client.deviceID = deviceID
}

func (client *Client) RemoteAddrString() string {
	return client.coapConn.RemoteAddr().String()
}

func (client *Client) Context() context.Context {
	return client.coapConn.Context()
}

// Close closes coap connection
func (client *Client) Close() error {
	if err := client.coapConn.Close(); err != nil {
		return fmt.Errorf("cannot close client: %w", err)
	}
	return nil
}

// OnClose action when coap connection was closed.
func (client *Client) OnClose() {
	log.Debugf("close client %v", client.coapConn.RemoteAddr())
}
