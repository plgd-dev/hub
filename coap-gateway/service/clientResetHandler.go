package service

import (
	gocoap "github.com/go-ocf/go-coap"
)

func clientResetHandler(s mux.ResponseWriter, req *message.Message, client *Client) {
	authCtx := client.loadAuthorizationContext()
	clientResetObservationHandler(s, req, client, authCtx.AuthorizationContext)
}
