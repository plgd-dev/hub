package coapconv

import (
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"google.golang.org/grpc/codes"
)

// ToGrpcCode converts coap.Code to grpc.Code
func ToGrpcCode(code coapCodes.Code, def codes.Code) codes.Code {
	switch code {
	case coapCodes.Empty:
		return codes.Unknown
	case coapCodes.Created:
		return codes.OK
	case coapCodes.Deleted:
		return codes.OK
	case coapCodes.Valid:
		return codes.OK
	case coapCodes.Changed:
		return codes.OK
	case coapCodes.Content:
		return codes.OK
	case coapCodes.Continue:
		return codes.OK
	case coapCodes.BadRequest:
		return codes.InvalidArgument
	case coapCodes.Unauthorized:
		return codes.Unauthenticated
	case coapCodes.BadOption:
		return codes.InvalidArgument
	case coapCodes.Forbidden:
		return codes.PermissionDenied
	case coapCodes.NotFound:
		return codes.NotFound
	case coapCodes.MethodNotAllowed:
		return codes.PermissionDenied
	case coapCodes.NotAcceptable:
		return codes.InvalidArgument
	case coapCodes.RequestEntityIncomplete:
		return codes.InvalidArgument
	case coapCodes.PreconditionFailed:
		return codes.FailedPrecondition
	case coapCodes.RequestEntityTooLarge:
		return codes.OutOfRange
	case coapCodes.UnsupportedMediaType:
		return codes.InvalidArgument
	case coapCodes.InternalServerError:
		return codes.Internal
	case coapCodes.NotImplemented:
		return codes.Unimplemented
	case coapCodes.BadGateway:
	case coapCodes.ServiceUnavailable:
		return codes.Unavailable
	case coapCodes.GatewayTimeout:
	case coapCodes.ProxyingNotSupported:
	}
	return def
}
