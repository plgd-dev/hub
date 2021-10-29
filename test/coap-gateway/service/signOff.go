package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/pkg/log"
)

// Sign-off
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signOffHandler(req *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %w", err), code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign off error: %w", err)
		}
	}

	if err := client.server.handler.SignOff(); err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}
