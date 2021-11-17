package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	coapgwService "github.com/plgd-dev/hub/coap-gateway/service"
	"github.com/plgd-dev/hub/pkg/log"
)

// Sign-Out
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signOutPostHandler(req *mux.Message, client *Client, signOut coapgwService.CoapSignInReq) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign out: %w", err), code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign out error: %w", err)
		}
	}

	if err := client.handler.SignOut(signOut); err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token, message.AppOcfCbor, []byte{0xA0}) // empty object
}
