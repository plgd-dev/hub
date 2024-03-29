package service

import (
	"bytes"
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
)

func (c *session) createErrorResponse(err error, token message.Token) *pool.Message {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	code := codes.BadRequest
	if ok {
		code = s.Code()
	}
	if coapgwMessage.IsTempError(err) {
		code = codes.ServiceUnavailable
		err = fmt.Errorf("temporary error: %w", err)
	}
	msg := c.server.messagePool.AcquireMessage(c.Context())
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	return msg
}
