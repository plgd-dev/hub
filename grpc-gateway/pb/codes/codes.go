package codes

import (
	"net/http"

	codes "google.golang.org/grpc/codes"
)

type Code codes.Code

const (
	// Accepted device accepts request and action will be proceed in future.
	Accepted Code = iota + 4096
	// MethodNotAllowed device refuse call the method.
	MethodNotAllowed
	// Created success status response code indicates that the request has succeeded and has led to the creation of a resource.
	Created

	// InvalidCode cannot determines result from device code.
	InvalidCode Code = iota + (2 * 4096)
)

var code2string = map[Code]string{
	Created:          "created",
	MethodNotAllowed: "methodNotAllowed",
	Accepted:         "accepted",
}

func (c Code) ToHTTPCode() int {
	switch c {
	case Code(codes.OK):
		return http.StatusOK
	case Code(codes.Canceled):
		return http.StatusRequestTimeout
	case Code(codes.Unknown):
		return http.StatusInternalServerError
	case Code(codes.InvalidArgument):
		return http.StatusBadRequest
	case Code(codes.DeadlineExceeded):
		return http.StatusGatewayTimeout
	case Code(codes.NotFound):
		return http.StatusNotFound
	case Code(codes.AlreadyExists):
		return http.StatusConflict
	case Code(codes.PermissionDenied):
		return http.StatusForbidden
	case Code(codes.Unauthenticated):
		return http.StatusUnauthorized
	case Code(codes.ResourceExhausted):
		return http.StatusTooManyRequests
	case Code(codes.FailedPrecondition):
		// Note, this deliberately doesn't translate to the similarly named '412 Precondition Failed' HTTP response status.
		return http.StatusBadRequest
	case Code(codes.Aborted):
		return http.StatusConflict
	case Code(codes.OutOfRange):
		return http.StatusBadRequest
	case Code(codes.Unimplemented):
		return http.StatusNotImplemented
	case Code(codes.Internal):
		return http.StatusInternalServerError
	case Code(codes.Unavailable):
		return http.StatusServiceUnavailable
	case Code(codes.DataLoss):
		return http.StatusInternalServerError
	case MethodNotAllowed:
		return http.StatusMethodNotAllowed
	case Created:
		return http.StatusCreated
	case Accepted:
		return http.StatusAccepted
	}
	return http.StatusInternalServerError
}

func (c Code) String() string {
	v, ok := code2string[c]
	if ok {
		return v
	}
	return codes.Code(c).String()
}
