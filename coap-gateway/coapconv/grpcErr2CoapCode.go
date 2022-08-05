package coapconv

import (
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

type Operation uint8

const (
	Create   Operation = 1
	Retrieve Operation = 2
	Update   Operation = 3
	Delete   Operation = 4
)

var grpcErrorCode2CoapCode map[codes.Code]coapCodes.Code = map[codes.Code]coapCodes.Code{
	codes.Canceled:           coapCodes.ServiceUnavailable,
	codes.Unknown:            coapCodes.InternalServerError,
	codes.InvalidArgument:    coapCodes.BadRequest,
	codes.DeadlineExceeded:   coapCodes.InternalServerError,
	codes.NotFound:           coapCodes.NotFound,
	codes.AlreadyExists:      coapCodes.InternalServerError,
	codes.PermissionDenied:   coapCodes.Forbidden,
	codes.ResourceExhausted:  coapCodes.InternalServerError,
	codes.FailedPrecondition: coapCodes.PreconditionFailed,
	codes.Aborted:            coapCodes.InternalServerError,
	codes.OutOfRange:         coapCodes.RequestEntityTooLarge,
	codes.Unimplemented:      coapCodes.NotImplemented,
	codes.Internal:           coapCodes.InternalServerError,
	codes.Unavailable:        coapCodes.ServiceUnavailable,
	codes.DataLoss:           coapCodes.InternalServerError,
	codes.Unauthenticated:    coapCodes.Unauthorized,
}

func grpcCode2CoapCode(statusCode codes.Code, operation Operation) coapCodes.Code {
	switch statusCode {
	case codes.OK:
		switch operation {
		case Update:
			return coapCodes.Changed
		case Retrieve:
			return coapCodes.Content
		case Create:
			return coapCodes.Created
		case Delete:
			return coapCodes.Deleted
		}
	default:
		if err, ok := grpcErrorCode2CoapCode[statusCode]; ok {
			return err
		}
	}
	return coapCodes.InternalServerError
}

func GrpcErr2CoapCode(err error, operation Operation) coapCodes.Code {
	grpcCode := codes.OK
	if err != nil {
		grpcCode = pkgGrpc.ErrToStatus(err).Code()
	}
	return grpcCode2CoapCode(grpcCode, operation)
}
