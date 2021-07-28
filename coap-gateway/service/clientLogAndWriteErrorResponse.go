package service

import (
	"bytes"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

func (client *Client) logAndWriteErrorResponse(err error, code codes.Code, token message.Token) {
	if err != nil {
		log.Errorf("%v", err)
	}
	msg := pool.AcquireMessage(client.coapConn.Context())
	defer pool.ReleaseMessage(msg)
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send error to %v: %v", getDeviceID(client), err)
	}
	decodeMsgToDebug(client, msg, "SEND-ERROR")
}
