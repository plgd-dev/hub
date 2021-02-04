package service

import (
	"github.com/plgd-dev/go-coap/v2/mux"
)

func clientResetHandler(req *mux.Message, client *Client) {
	clientResetObservationHandler(req, client)
}
