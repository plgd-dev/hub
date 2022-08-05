package service

import (
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

func clientResetHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	return clientResetObservationHandler(req, client)
}
