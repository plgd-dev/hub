package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
)

func (client *Client) logAndWriteErrorResponse(req *mux.Message, err error, code codes.Code, token message.Token) {
	if coapgwMessage.IsTempError(err) {
		code = codes.ServiceUnavailable
		err = fmt.Errorf("temporary error: %w", err)
	}
	msg, cleanUp := coapgwMessage.GetErrorResponse(client.coapConn.Context(), client.server.messagePool, code, token, err)
	defer cleanUp()
	client.sendCoapResponse(req, msg)
}
