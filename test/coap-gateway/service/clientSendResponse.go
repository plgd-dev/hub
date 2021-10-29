package service

import (
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	coapgwMessage "github.com/plgd-dev/hub/coap-gateway/service/message"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
)

func (client *Client) sendResponse(code coapCodes.Code, token message.Token, contentFormat message.MediaType, payload []byte) {
	msg, cleanUp := coapgwMessage.GetResponse(client.coapConn.Context(), code, token, contentFormat, payload)
	defer cleanUp()
	err := client.coapConn.WriteMessage(msg)
	if err != nil {
		if !kitNetGrpc.IsContextCanceled(err) {
			log.Errorf("cannot send reply to %v: %w", client.GetDeviceID(), err)
		}
	}
	decodeMsgToDebug(client, msg, "SEND-RESPONSE")
}
