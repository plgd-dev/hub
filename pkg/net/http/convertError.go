package http

import (
	"errors"
	netHttp "net/http"

	coapStatus "github.com/plgd-dev/go-coap/v2/message/status"
	"github.com/plgd-dev/kit/v2/coapconv"
	"github.com/plgd-dev/kit/v2/grpcconv"
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
	var gErr grpcErr
	if errors.As(err, &gErr) {
		return grpcconv.ToHTTPCode(gErr.GRPCStatus().Code(), def)
	}
	var sErr sdkErr
	if errors.As(err, &sErr) {
		return grpcconv.ToHTTPCode(sErr.GetCode(), def)
	}
	return def
}

// ErrToStatus converts err to http.Status.
func ErrToStatus(err error) int {
	return ErrToStatusWithDef(err, netHttp.StatusInternalServerError)
}
