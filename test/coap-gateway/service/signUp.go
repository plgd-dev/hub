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

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpPostHandler(r *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), code, r.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign up error: %w", err)
		}
	}

	var signUp coapgwService.CoapSignUpRequest
	if err := cbor.ReadFrom(r.Body, &signUp); err != nil {
		logErrorAndCloseClient(err, coapCodes.BadRequest)
		return
	}

	client.SetDeviceID(signUp.DeviceID)

	resp, err := client.handler.SignUp(signUp)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	accept := coapconv.GetAccept(r.Options)
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

	client.sendResponse(coapCodes.Changed, r.Token, accept, out)
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpHandler(r *mux.Message, client *Client) {
	switch r.Code {
	case coapCodes.POST:
		signUpPostHandler(r, client)
	case coapCodes.DELETE:
		signOffHandler(r, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, r.Token)
	}
}
