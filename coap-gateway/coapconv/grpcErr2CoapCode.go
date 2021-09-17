package coapconv

import (
	pkgGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"google.golang.org/grpc/codes"
)

type Operation uint8

const Create Operation = 1
const Retrieve Operation = 2
const Update Operation = 3
const Delete Operation = 4

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
	case codes.Canceled:
		return coapCodes.ServiceUnavailable
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

func GrpcErr2CoapCode(err error, operation Operation) coapCodes.Code {
	grpcCode := codes.OK
	if err != nil {
		grpcCode = pkgGrpc.ErrToStatus(err).Code()
	}
	return grpcCode2CoapCode(grpcCode, operation)
}
