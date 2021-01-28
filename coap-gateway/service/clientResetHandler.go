package service

import (
	"github.com/plgd-dev/go-coap/v2/mux"
)

func clientResetHandler(req *mux.Message, client *Client) {
	authCtx := client.loadAuthorizationContext()
	clientResetObservationHandler(req, client, authCtx.AuthorizationContext)
}
