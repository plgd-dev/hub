package service

import (
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
)

func (client *Client) sendCoapResponse(req *mux.Message, msg *pool.Message) {
	err := client.coapConn.WriteMessage(msg)
	if err != nil {
		if !kitNetGrpc.IsContextCanceled(err) {
			client.Errorf("cannot send reply to %v: %w", getDeviceID(client), err)
		}
	}
	client.logClientRequest(req, msg)
}

func (client *Client) sendResponse(req *mux.Message, code coapCodes.Code, token message.Token, contentFormat message.MediaType, payload []byte) {
	msg, cleanUp := coapgwMessage.GetResponse(client.coapConn.Context(), client.server.messagePool, code, token, contentFormat, payload)
	defer cleanUp()
	client.sendCoapResponse(req, msg)
}
