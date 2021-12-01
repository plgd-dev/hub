package service

import (
	"fmt"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/coap-gateway/service"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

func refreshTokenPostHandler(req *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token: %w", err), code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("refresh token error: %w", err)
		}
	}

	var r service.CoapRefreshTokenReq
	err := cbor.ReadFrom(req.Body, &r)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.BadRequest)
		return
	}

	resp, err := client.server.handler.RefreshToken(r)
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

// RefreshToken
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.tokenrefresh.swagger.json
func refreshTokenHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		refreshTokenPostHandler(req, client)
		return
	}
	client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
}
