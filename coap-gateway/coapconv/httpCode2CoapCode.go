package coapconv

import (
	"net/http"

	coapCodes "github.com/go-ocf/go-coap/codes"
)

func HttpCode2CoapCode(statusCode int, method coapCodes.Code) coapCodes.Code {
	switch statusCode {
	//1xx
	case http.StatusContinue:
		return coapCodes.Continue
	case http.StatusSwitchingProtocols:
	case http.StatusProcessing:

	//2xx
	case http.StatusOK:
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
	case http.StatusCreated:
		return coapCodes.Created
	case http.StatusAccepted:
	case http.StatusNonAuthoritativeInfo:
	case http.StatusNoContent:
	case http.StatusResetContent:
	case http.StatusPartialContent:
	case http.StatusMultiStatus:
	case http.StatusAlreadyReported:
	case http.StatusIMUsed:

	//3xx
	case http.StatusMultipleChoices:
	case http.StatusMovedPermanently:
	case http.StatusFound:
	case http.StatusSeeOther:
	case http.StatusNotModified:
	case http.StatusUseProxy:
	case http.StatusTemporaryRedirect:
	case http.StatusPermanentRedirect:

	//4xx
	case http.StatusBadRequest:
		return coapCodes.BadRequest
	case http.StatusUnauthorized:
		return coapCodes.Unauthorized
	case http.StatusPaymentRequired:
	case http.StatusForbidden:
		return coapCodes.Forbidden
	case http.StatusNotFound:
		return coapCodes.NotFound
	case http.StatusMethodNotAllowed:
		return coapCodes.MethodNotAllowed
	case http.StatusNotAcceptable:
		return coapCodes.NotAcceptable
	case http.StatusProxyAuthRequired:
	case http.StatusRequestTimeout:
	case http.StatusConflict:
	case http.StatusGone:
	case http.StatusLengthRequired:
	case http.StatusPreconditionFailed:
		return coapCodes.PreconditionFailed
	case http.StatusRequestEntityTooLarge:
		return coapCodes.RequestEntityTooLarge
	case http.StatusRequestURITooLong:
	case http.StatusUnsupportedMediaType:
		return coapCodes.UnsupportedMediaType
	case http.StatusRequestedRangeNotSatisfiable:
	case http.StatusExpectationFailed:
	case http.StatusTeapot:
	case http.StatusUnprocessableEntity:
	case http.StatusLocked:
	case http.StatusFailedDependency:
	case http.StatusUpgradeRequired:
	case http.StatusPreconditionRequired:
	case http.StatusTooManyRequests:
	case http.StatusRequestHeaderFieldsTooLarge:
	case http.StatusUnavailableForLegalReasons:

	//5xx
	case http.StatusInternalServerError:
	case http.StatusNotImplemented:
		return coapCodes.NotImplemented
	case http.StatusBadGateway:
		return coapCodes.BadGateway
	case http.StatusServiceUnavailable:
		return coapCodes.ServiceUnavailable
	case http.StatusGatewayTimeout:
		return coapCodes.GatewayTimeout
	case http.StatusHTTPVersionNotSupported:
	case http.StatusVariantAlsoNegotiates:
	case http.StatusInsufficientStorage:
	case http.StatusLoopDetected:
	case http.StatusNotExtended:
	case http.StatusNetworkAuthenticationRequired:
	}
	return coapCodes.InternalServerError
}
