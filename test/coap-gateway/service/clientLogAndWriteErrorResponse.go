package service

import (
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (c *Client) logAndWriteErrorResponse(err error, code codes.Code, token message.Token) {
	msg, cleanUp := coapgwMessage.GetErrorResponse(c.Context(), pool.New(0, 0), code, token, err)
	defer cleanUp()
	err = c.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send error to %v: %w", c.GetDeviceID(), err)
	}
	decodeMsgToDebug(c, msg, "SEND-ERROR")
}
