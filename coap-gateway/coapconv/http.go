package coapconv

import (
	"net/http"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
)

var coapToHTTP = map[coapCodes.Code]int{
	coapCodes.Created:                 http.StatusCreated,
	coapCodes.Deleted:                 http.StatusOK,
	coapCodes.Valid:                   http.StatusAccepted,
	coapCodes.Changed:                 http.StatusOK,
	coapCodes.Content:                 http.StatusOK,
	coapCodes.Continue:                http.StatusContinue,
	coapCodes.BadRequest:              http.StatusBadRequest,
	coapCodes.Unauthorized:            http.StatusUnauthorized,
	coapCodes.BadOption:               http.StatusBadRequest,
	coapCodes.Forbidden:               http.StatusForbidden,
	coapCodes.NotFound:                http.StatusNotFound,
	coapCodes.MethodNotAllowed:        http.StatusMethodNotAllowed,
	coapCodes.NotAcceptable:           http.StatusNotAcceptable,
	coapCodes.RequestEntityIncomplete: http.StatusBadRequest,
	coapCodes.PreconditionFailed:      http.StatusPreconditionFailed,
	coapCodes.RequestEntityTooLarge:   http.StatusRequestEntityTooLarge,
	coapCodes.UnsupportedMediaType:    http.StatusUnsupportedMediaType,
	coapCodes.InternalServerError:     http.StatusInternalServerError,
	coapCodes.NotImplemented:          http.StatusNotImplemented,
	coapCodes.BadGateway:              http.StatusBadGateway,
	coapCodes.ServiceUnavailable:      http.StatusServiceUnavailable,
	coapCodes.GatewayTimeout:          http.StatusGatewayTimeout,
}

// ToHTTPCode converts coap.Code to http.Status
func ToHTTPCode(code coapCodes.Code, def int) int {
	if val, ok := coapToHTTP[code]; ok {
		return val
	}
	return def
}
