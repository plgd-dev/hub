package message

import (
	"bytes"
	"context"
	"strings"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

func GetResponse(ctx context.Context, messagePool *pool.Pool, code codes.Code, token message.Token, contentFormat message.MediaType, payload []byte) (*pool.Message, func()) {
	msg := messagePool.AcquireMessage(ctx)
	msg.SetCode(code)
	msg.SetToken(token)
	if len(payload) > 0 {
		msg.SetContentFormat(contentFormat)
		msg.SetBody(bytes.NewReader(payload))
	}
	return msg, func() {
		messagePool.ReleaseMessage(msg)
	}
}

func IsTempError(err error) bool {
	switch {
	// TODO: We could optimize this by using error.Is to avoid string comparison.
	case strings.Contains(err.Error(), "connect: connection refused"),
		strings.Contains(err.Error(), "i/o timeout"),
		strings.Contains(err.Error(), "TLS handshake timeout"),
		strings.Contains(err.Error(), `http2:`), // any error at http2 protocol is considered as temporary error
		strings.Contains(err.Error(), `write: broken pipe`),
		strings.Contains(err.Error(), context.DeadlineExceeded.Error()),
		strings.Contains(err.Error(), context.Canceled.Error()):
		return true
	}
	return false
}

func GetErrorResponse(ctx context.Context, messagePool *pool.Pool, code codes.Code, token message.Token, err error) (*pool.Message, func()) {
	msg := messagePool.AcquireMessage(ctx)
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	return msg, func() {
		messagePool.ReleaseMessage(msg)
	}
}
