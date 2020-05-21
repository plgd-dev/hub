package service

import (
	"bytes"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/go-ocf/kit/log"
)

func logAndWriteErrorResponse(err error, client *Client, code codes.Code, token message.Token) {
	msg := pool.AcquireMessage(client.coapConn.Context())
	if err != nil {
		log.Errorf("%v", err)
	}
	defer pool.ReleaseMessage(msg)
	msg.SetCode(code)
	msg.SetToken(token)
	if client != nil && client.server.SendErrorTextInResponse {
		msg.SetContentFormat(message.TextPlain)
		msg.SetBody(bytes.NewReader([]byte(err.Error())))
	} else {
		msg.SetContentFormat(message.AppOcfCbor)
		msg.SetBody(bytes.NewReader([]byte{0xA0})) // empty object
	}
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send error to %v: %v", getDeviceID(client), err)
	}
	decodeMsgToDebug(client, msg, "SEND-ERROR")
}
