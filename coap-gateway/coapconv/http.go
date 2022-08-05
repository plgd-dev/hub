package coapconv

import (
	"net/http"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
)

// ToHTTPCode converts coap.Code to http.Status
func ToHTTPCode(code coapCodes.Code, def int) int {
	switch code {
	case coapCodes.Empty:
	case coapCodes.Created:
		return http.StatusCreated
	case coapCodes.Deleted:
		return http.StatusOK
	case coapCodes.Valid:
		return http.StatusAccepted
	case coapCodes.Changed:
		return http.StatusOK
	case coapCodes.Content:
		return http.StatusOK
	case coapCodes.Continue:
		return http.StatusContinue
	case coapCodes.BadRequest:
		return http.StatusBadRequest
	case coapCodes.Unauthorized:
		return http.StatusUnauthorized
	case coapCodes.BadOption:
		return http.StatusBadRequest
	case coapCodes.Forbidden:
		return http.StatusForbidden
	case coapCodes.NotFound:
		return http.StatusNotFound
	case coapCodes.MethodNotAllowed:
		return http.StatusMethodNotAllowed
	case coapCodes.NotAcceptable:
		return http.StatusNotAcceptable
	case coapCodes.RequestEntityIncomplete:
		return http.StatusBadRequest
	case coapCodes.PreconditionFailed:
		return http.StatusPreconditionFailed
	case coapCodes.RequestEntityTooLarge:
		return http.StatusRequestEntityTooLarge
	case coapCodes.UnsupportedMediaType:
		return http.StatusUnsupportedMediaType
	case coapCodes.InternalServerError:
		return http.StatusInternalServerError
	case coapCodes.NotImplemented:
		return http.StatusNotImplemented
	case coapCodes.BadGateway:
		return http.StatusBadGateway
	case coapCodes.ServiceUnavailable:
		return http.StatusServiceUnavailable
	case coapCodes.GatewayTimeout:
		return http.StatusGatewayTimeout
	case coapCodes.ProxyingNotSupported:
	}
	return def
}
