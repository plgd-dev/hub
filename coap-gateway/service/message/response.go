package message

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"golang.org/x/oauth2"
	grpcCodes "google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
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

func isGrpcTempError(err error) (ok bool, result bool) {
	var grpcStatus interface {
		GRPCStatus() *grpcStatus.Status
	}
	if errors.As(err, &grpcStatus) {
		switch grpcStatus.GRPCStatus().Code() {
		case grpcCodes.PermissionDenied,
			grpcCodes.Unauthenticated,
			grpcCodes.NotFound,
			grpcCodes.AlreadyExists,
			grpcCodes.InvalidArgument:
			return true, false
		}
		return true, true
	}
	return false, false
}

func isOauth2TempError(err error) (ok bool, result bool) {
	oauth2Err := &oauth2.RetrieveError{}
	if errors.As(err, &oauth2Err) {
		if oauth2Err.Response != nil {
			switch oauth2Err.Response.StatusCode {
			case
				http.StatusBadRequest,
				http.StatusConflict,
				http.StatusNotFound,
				http.StatusForbidden,
				http.StatusUnauthorized:
				return true, false
			}
		}
		return true, true
	}
	return false, false
}

// IsTempError returns true if error is temporary. Only certain errors are not considered as temporary errors.
func IsTempError(err error) bool {
	if err == nil {
		return false
	}
	var isTemporary interface {
		Temporary() bool
	}
	if errors.As(err, &isTemporary) && isTemporary.Temporary() {
		return true
	}
	var isTimeout interface {
		Timeout() bool
	}
	if errors.As(err, &isTimeout) && isTimeout.Timeout() {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	ok, temp := isGrpcTempError(err)
	if ok {
		return temp
	}

	ok, temp = isOauth2TempError(err)
	if ok {
		return temp
	}

	switch {
	// TODO: We could optimize this by using error.Is to avoid string comparison.
	case strings.Contains(err.Error(), "connect: connection refused"),
		strings.Contains(err.Error(), "i/o timeout"),
		strings.Contains(err.Error(), "TLS handshake timeout"),
		strings.Contains(err.Error(), `http2:`), // any error at http2 protocol is considered as temporary error
		strings.Contains(err.Error(), `write: broken pipe`),
		strings.Contains(err.Error(), `request canceled while waiting for connection`),
		strings.Contains(err.Error(), `authentication handshake failed`):
		return true
	}

	if _, ok := status.FromError(err); ok {
		// coap status code is not temporary
		return false
	}
	return true
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
