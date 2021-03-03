package coapconv

import (
	"testing"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/valyala/fasthttp"
)

func TestHttpCode2CoapCode(t *testing.T) {
	tbl := []struct {
		name         string
		inStatusCode int
		inCoapCode   coapCodes.Code
		out          coapCodes.Code
	}{
		{"Continue", fasthttp.StatusContinue, coapCodes.Empty, coapCodes.Continue},
		{"fasthttp.StatusSwitchingProtocols", fasthttp.StatusSwitchingProtocols, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusProcessing", fasthttp.StatusProcessing, coapCodes.Empty, coapCodes.InternalServerError},

		//2xx
		{"fasthttp.StatusOK", fasthttp.StatusOK, coapCodes.POST, coapCodes.Changed},
		{"fasthttp.StatusOK", fasthttp.StatusOK, coapCodes.GET, coapCodes.Content},
		{"fasthttp.StatusOK", fasthttp.StatusOK, coapCodes.DELETE, coapCodes.Deleted},

		{"fasthttp.StatusCreated", fasthttp.StatusCreated, coapCodes.Empty, coapCodes.Created},
		{"fasthttp.StatusAccepted", fasthttp.StatusAccepted, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusNonAuthoritativeInfo", fasthttp.StatusNonAuthoritativeInfo, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusNoContent", fasthttp.StatusNoContent, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusResetContent", fasthttp.StatusResetContent, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusPartialContent", fasthttp.StatusPartialContent, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusMultiStatus", fasthttp.StatusMultiStatus, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusAlreadyReported", fasthttp.StatusAlreadyReported, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusIMUsed", fasthttp.StatusIMUsed, coapCodes.Empty, coapCodes.InternalServerError},

		//3xx
		{"fasthttp.StatusMultipleChoices", fasthttp.StatusMultipleChoices, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusMovedPermanently", fasthttp.StatusMovedPermanently, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusFound", fasthttp.StatusFound, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusSeeOther", fasthttp.StatusSeeOther, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusNotModified", fasthttp.StatusNotModified, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusUseProxy", fasthttp.StatusUseProxy, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusTemporaryRedirect", fasthttp.StatusTemporaryRedirect, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusPermanentRedirect", fasthttp.StatusPermanentRedirect, coapCodes.Empty, coapCodes.InternalServerError},

		//4xx
		{"fasthttp.StatusBadRequest", fasthttp.StatusBadRequest, coapCodes.Empty, coapCodes.BadRequest},
		{"fasthttp.StatusUnauthorized", fasthttp.StatusUnauthorized, coapCodes.Empty, coapCodes.Unauthorized},
		{"fasthttp.StatusPaymentRequired", fasthttp.StatusPaymentRequired, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusForbidden", fasthttp.StatusForbidden, coapCodes.Empty, coapCodes.Forbidden},
		{"fasthttp.StatusNotFound", fasthttp.StatusNotFound, coapCodes.Empty, coapCodes.NotFound},
		{"fasthttp.StatusMethodNotAllowed", fasthttp.StatusMethodNotAllowed, coapCodes.Empty, coapCodes.MethodNotAllowed},
		{"fasthttp.StatusNotAcceptable", fasthttp.StatusNotAcceptable, coapCodes.Empty, coapCodes.NotAcceptable},
		{"fasthttp.StatusProxyAuthRequired", fasthttp.StatusProxyAuthRequired, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusRequestTimeout", fasthttp.StatusRequestTimeout, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusConflict", fasthttp.StatusConflict, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusGone", fasthttp.StatusGone, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusLengthRequired", fasthttp.StatusLengthRequired, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusPreconditionFailed", fasthttp.StatusPreconditionFailed, coapCodes.Empty, coapCodes.PreconditionFailed},
		{"fasthttp.StatusRequestEntityTooLarge", fasthttp.StatusRequestEntityTooLarge, coapCodes.Empty, coapCodes.RequestEntityTooLarge},
		{"fasthttp.StatusRequestURITooLong", fasthttp.StatusRequestURITooLong, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusUnsupportedMediaType", fasthttp.StatusUnsupportedMediaType, coapCodes.Empty, coapCodes.UnsupportedMediaType},
		{"fasthttp.StatusRequestedRangeNotSatisfiable", fasthttp.StatusRequestedRangeNotSatisfiable, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusExpectationFailed", fasthttp.StatusExpectationFailed, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusTeapot", fasthttp.StatusTeapot, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusUnprocessableEntity", fasthttp.StatusUnprocessableEntity, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusLocked", fasthttp.StatusLocked, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusFailedDependency", fasthttp.StatusFailedDependency, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusUpgradeRequired", fasthttp.StatusUpgradeRequired, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusPreconditionRequired", fasthttp.StatusPreconditionRequired, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusTooManyRequests", fasthttp.StatusTooManyRequests, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusRequestHeaderFieldsTooLarge", fasthttp.StatusRequestHeaderFieldsTooLarge, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusUnavailableForLegalReasons", fasthttp.StatusUnavailableForLegalReasons, coapCodes.Empty, coapCodes.InternalServerError},

		//5xx
		{"fasthttp.StatusInternalServerError", fasthttp.StatusInternalServerError, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusNotImplemented", fasthttp.StatusNotImplemented, coapCodes.Empty, coapCodes.NotImplemented},
		{"fasthttp.StatusBadGateway", fasthttp.StatusBadGateway, coapCodes.Empty, coapCodes.BadGateway},
		{"fasthttp.StatusServiceUnavailable", fasthttp.StatusServiceUnavailable, coapCodes.Empty, coapCodes.ServiceUnavailable},
		{"fasthttp.StatusGatewayTimeout", fasthttp.StatusGatewayTimeout, coapCodes.Empty, coapCodes.GatewayTimeout},
		{"fasthttp.StatusHTTPVersionNotSupported", fasthttp.StatusHTTPVersionNotSupported, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusVariantAlsoNegotiates", fasthttp.StatusVariantAlsoNegotiates, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusInsufficientStorage", fasthttp.StatusInsufficientStorage, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusLoopDetected", fasthttp.StatusLoopDetected, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusNotExtended", fasthttp.StatusNotExtended, coapCodes.Empty, coapCodes.InternalServerError},
		{"fasthttp.StatusNetworkAuthenticationRequired", fasthttp.StatusNetworkAuthenticationRequired, coapCodes.Empty, coapCodes.InternalServerError},
	}
	for _, e := range tbl {
		testCode := func(t *testing.T) {
			code := HttpCode2CoapCode(e.inStatusCode, e.inCoapCode)
			if e.out != code {
				t.Errorf("Unexpected code(%v) returned, expected %v", code, e.out)
			}
		}
		t.Run(e.name, testCode)
	}
}
