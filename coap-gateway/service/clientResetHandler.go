package service

import (
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

func clientResetHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	return clientResetObservationHandler(req, client)
}
