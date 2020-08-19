package coapconv

import (
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"google.golang.org/grpc/codes"
)

func GrpcCode2CoapCode(statusCode codes.Code, method coapCodes.Code) coapCodes.Code {
	switch statusCode {
	case codes.OK:
		switch method {
		case coapCodes.POST:
			return coapCodes.Changed
		case coapCodes.GET:
			return coapCodes.Content
		case coapCodes.PUT:
			return coapCodes.Created
		case coapCodes.DELETE:
			return coapCodes.Deleted
		}
	case codes.Canceled:
		return coapCodes.Empty
	case codes.Unknown:
		return coapCodes.InternalServerError
	case codes.InvalidArgument:
		return coapCodes.BadRequest
	case codes.DeadlineExceeded:
		return coapCodes.InternalServerError
	case codes.NotFound:
		return coapCodes.NotFound
	case codes.AlreadyExists:
		return coapCodes.InternalServerError
	case codes.PermissionDenied:
		return coapCodes.Forbidden
	case codes.ResourceExhausted:
		return coapCodes.InternalServerError
	case codes.FailedPrecondition:
		return coapCodes.PreconditionFailed
	case codes.Aborted:
		return coapCodes.InternalServerError
	case codes.OutOfRange:
		return coapCodes.RequestEntityTooLarge
	case codes.Unimplemented:
		return coapCodes.NotImplemented
	case codes.Internal:
		return coapCodes.InternalServerError
	case codes.Unavailable:
		return coapCodes.ServiceUnavailable
	case codes.DataLoss:
		return coapCodes.InternalServerError
	case codes.Unauthenticated:
		return coapCodes.Unauthorized
	}
	return coapCodes.InternalServerError
}
