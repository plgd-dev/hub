package service

import (
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (client *Client) logAndWriteErrorResponse(err error, code codes.Code, token message.Token) {
	msg, cleanUp := coapgwMessage.LogAndGetErrorResponse(client.coapConn.Context(), client.server.messagePool, code, token, err)
	defer cleanUp()
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send error to %v: %w", getDeviceID(client), err)
	}
	decodeMsgToDebug(client, msg, "SEND-ERROR")
}
