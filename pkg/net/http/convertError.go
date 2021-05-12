package http

import (
	"errors"
	netHttp "net/http"

	coapStatus "github.com/plgd-dev/go-coap/v2/message/status"
	"github.com/plgd-dev/kit/coapconv"
	"github.com/plgd-dev/kit/grpcconv"
	grpcCodes "google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

type grpcErr interface {
	GRPCStatus() *grpcStatus.Status
}

type sdkErr interface {
	GetCode() grpcCodes.Code
}

// ErrToStatusWithDef converts err with default http.Status(for unknown conversion) to http.Status.
func ErrToStatusWithDef(err error, def int) int {
	if err == nil {
		return netHttp.StatusOK
	}
	var coapStatus coapStatus.Status
	if errors.As(err, &coapStatus) {
		return coapconv.ToHTTPCode(coapStatus.Message().Code, def)
	}
	var grpcErr grpcErr
	if errors.As(err, &grpcErr) {
		return grpcconv.ToHTTPCode(grpcErr.GRPCStatus().Code(), def)
	}
	var sdkErr sdkErr
	if errors.As(err, &sdkErr) {
		return grpcconv.ToHTTPCode(sdkErr.GetCode(), def)
	}
	return def
}

// ErrToStatus converts err to http.Status.
func ErrToStatus(err error) int {
	return ErrToStatusWithDef(err, netHttp.StatusInternalServerError)
}
