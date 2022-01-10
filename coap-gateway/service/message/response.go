package message

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/pkg/log"
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

func isTempError(err error) bool {
	switch {
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

func LogAndGetErrorResponse(ctx context.Context, messagePool *pool.Pool, code codes.Code, token message.Token, err error) (*pool.Message, func()) {
	if isTempError(err) {
		code = codes.ServiceUnavailable
		err = fmt.Errorf("temporary error: %w", err)
	}
	if err != nil {
		log.Errorf("%w", err)
	}
	return GetErrorResponse(ctx, messagePool, code, token, err)
}
