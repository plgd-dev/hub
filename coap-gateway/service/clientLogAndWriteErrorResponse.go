package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

func isTempError(err error) bool {
	switch {
	case strings.Contains(err.Error(), "connect: connection refused"),
		strings.Contains(err.Error(), "i/o timeout"),
		strings.Contains(err.Error(), "TLS handshake timeout"),
		strings.Contains(err.Error(), `http2: client connection force closed via ClientConn.Close`),
		strings.Contains(err.Error(), `write: broken pipe`),
		strings.Contains(err.Error(), context.DeadlineExceeded.Error()),
		strings.Contains(err.Error(), context.Canceled.Error()):
		return true
	}
	return false
}

func (client *Client) logAndWriteErrorResponse(err error, code codes.Code, token message.Token) {
	if isTempError(err) {
		code = codes.ServiceUnavailable
		err = fmt.Errorf("temporary error: %w", err)
	}

	if err != nil {
		log.Errorf("%w", err)
	}
	msg := pool.AcquireMessage(client.coapConn.Context())
	defer pool.ReleaseMessage(msg)
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send error to %v: %w", getDeviceID(client), err)
	}
	decodeMsgToDebug(client, msg, "SEND-ERROR")
}
