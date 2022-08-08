package service

import (
	"fmt"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(req *mux.Message, client *Client, signIn coapgwService.CoapSignInReq) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign in error: %w", err)
		}
	}

	resp, err := client.handler.SignIn(signIn)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	accept := coapconv.GetAccept(req.Options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}
	out, err := encode(resp)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token, accept, out)
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInHandler(req *mux.Message, client *Client) {
	if req.Code == coapCodes.POST {
		var r coapgwService.CoapSignInReq
		err := cbor.ReadFrom(req.Body, &r)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapCodes.BadRequest, req.Token)
			return
		}
		if r.Login {
			signInPostHandler(req, client, r)
		} else {
			signOutPostHandler(req, client, r)
		}
		return
	}
	client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, req.Token)
}
