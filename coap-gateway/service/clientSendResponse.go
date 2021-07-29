package service

import (
	"bytes"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

func (client *Client) sendResponse(code coapCodes.Code, token message.Token, contentFormat message.MediaType, payload []byte) {
	msg := pool.AcquireMessage(client.coapConn.Context())
	defer pool.ReleaseMessage(msg)
	msg.SetCode(code)
	msg.SetToken(token)
	if len(payload) > 0 {
		msg.SetContentFormat(contentFormat)
		msg.SetBody(bytes.NewReader(payload))
	}
	err := client.coapConn.WriteMessage(msg)
	if err != nil {
		if !kitNetGrpc.IsContextCanceled(err) {
			log.Errorf("cannot send reply to %v: %w", getDeviceID(client), err)
		}
	}
	decodeMsgToDebug(client, msg, "SEND-RESPONSE")
}
