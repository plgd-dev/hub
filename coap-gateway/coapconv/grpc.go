package coapconv

import (
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"google.golang.org/grpc/codes"
)

var mapToGrpcCode = map[coapCodes.Code]codes.Code{
	coapCodes.Empty:                   codes.Unknown,
	coapCodes.Created:                 codes.OK,
	coapCodes.Deleted:                 codes.OK,
	coapCodes.Valid:                   codes.OK,
	coapCodes.Changed:                 codes.OK,
	coapCodes.Content:                 codes.OK,
	coapCodes.Continue:                codes.OK,
	coapCodes.BadRequest:              codes.InvalidArgument,
	coapCodes.Unauthorized:            codes.Unauthenticated,
	coapCodes.BadOption:               codes.InvalidArgument,
	coapCodes.Forbidden:               codes.PermissionDenied,
	coapCodes.NotFound:                codes.NotFound,
	coapCodes.MethodNotAllowed:        codes.PermissionDenied,
	coapCodes.NotAcceptable:           codes.InvalidArgument,
	coapCodes.RequestEntityIncomplete: codes.InvalidArgument,
	coapCodes.PreconditionFailed:      codes.FailedPrecondition,
	coapCodes.RequestEntityTooLarge:   codes.OutOfRange,
	coapCodes.UnsupportedMediaType:    codes.InvalidArgument,
	coapCodes.InternalServerError:     codes.Internal,
	coapCodes.NotImplemented:          codes.Unimplemented,
	coapCodes.BadGateway:              codes.Unavailable,
	coapCodes.ServiceUnavailable:      codes.Unavailable,
	coapCodes.GatewayTimeout:          codes.Unavailable,
	coapCodes.ProxyingNotSupported:    codes.Unimplemented,
}

// ToGrpcCode converts coap.Code to grpc.Code
func ToGrpcCode(code coapCodes.Code, def codes.Code) codes.Code {
	if c, ok := mapToGrpcCode[code]; ok {
		return c
	}
	return def
}
