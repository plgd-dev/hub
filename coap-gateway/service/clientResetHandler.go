package service

import (
	"github.com/plgd-dev/go-coap/v2/mux"
)

func clientResetHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	authCtx := client.loadAuthorizationContext()
	clientResetObservationHandler(s, req, client, authCtx.AuthorizationContext)
}
