package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

// Sign-Out
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signOutPostHandler(req *mux.Message, client *Client, signOut coapgwService.CoapSignInReq) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign out: %w", err), code, req.Token())
		if client.handler == nil || client.handler.CloseOnError() {
			if err := client.Close(); err != nil {
				log.Errorf("sign out error: %w", err)
			}
		}
	}

	err := client.handler.SignOut(signOut)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token(), message.AppOcfCbor, []byte{0xA0}) // empty object
}
