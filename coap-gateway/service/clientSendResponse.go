package service

import (
	"bytes"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

func (c *Client) createResponse(code coapCodes.Code, token message.Token, contentFormat message.MediaType, payload []byte) *pool.Message {
	msg := c.server.messagePool.AcquireMessage(c.coapConn.Context())
	msg.SetCode(code)
	msg.SetToken(token)
	if len(payload) > 0 {
		msg.SetContentFormat(contentFormat)
		msg.SetBody(bytes.NewReader(payload))
	}
	return msg
}
