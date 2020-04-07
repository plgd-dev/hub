package service

import (
	gocoap "github.com/go-ocf/go-coap"
)

func clientResetHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	authCtx := client.loadAuthorizationContext()
	clientResetObservationHandler(s, req, client, authCtx.AuthorizationContext)
}
