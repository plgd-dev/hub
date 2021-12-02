package service

import (
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	coapgwMessage "github.com/plgd-dev/hub/coap-gateway/service/message"
	"github.com/plgd-dev/hub/pkg/log"
)

func (client *Client) logAndWriteErrorResponse(err error, code codes.Code, token message.Token) {
	msg, cleanUp := coapgwMessage.LogAndGetErrorResponse(client.Context(), pool.New(0, 0), code, token, err)
	defer cleanUp()
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send error to %v: %w", client.GetDeviceID(), err)
	}
	decodeMsgToDebug(client, msg, "SEND-ERROR")
}
