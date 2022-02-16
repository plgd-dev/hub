package http

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	netHttp "net/http"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	return n, err
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	writer, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("not supported by the underlying writer")
	}
	return writer.Hijack()
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func DefaultCodeToLevel(code int, logger log.Logger) func(args ...interface{}) {
	switch code {
	case http.StatusContinue:
		fallthrough
	case http.StatusSwitchingProtocols:
		fallthrough
	case http.StatusProcessing:
		fallthrough
	case http.StatusEarlyHints:

	case http.StatusOK:
		fallthrough
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		fallthrough
	case http.StatusNonAuthoritativeInfo:
		fallthrough
	case http.StatusNoContent:
		fallthrough
	case http.StatusResetContent:
		fallthrough
	case http.StatusPartialContent:
		fallthrough
	case http.StatusMultiStatus:
		fallthrough
	case http.StatusAlreadyReported:
		fallthrough
	case http.StatusIMUsed:
		fallthrough
	case http.StatusMultipleChoices:
		fallthrough
	case http.StatusMovedPermanently:
		fallthrough
	case http.StatusFound:
		fallthrough
	case http.StatusSeeOther:
		fallthrough
	case http.StatusNotModified:
		fallthrough
	case http.StatusUseProxy:
		fallthrough
	case http.StatusTemporaryRedirect:
		fallthrough
	case http.StatusPermanentRedirect:
		return logger.Debug

	case http.StatusBadRequest:
		return logger.Debug
	case http.StatusUnauthorized:
		return logger.Debug
	case http.StatusPaymentRequired:
		return logger.Debug
	case http.StatusForbidden:
		return logger.Warn
	case http.StatusNotFound:
		return logger.Debug
	case http.StatusMethodNotAllowed:
		return logger.Debug
	case http.StatusNotAcceptable:
		return logger.Debug
	case http.StatusProxyAuthRequired:
		return logger.Debug
	case http.StatusRequestTimeout:
		return logger.Debug
	case http.StatusConflict:
		return logger.Debug
	case http.StatusGone:
		return logger.Debug
	case http.StatusLengthRequired:
		return logger.Debug
	case http.StatusPreconditionFailed:
		return logger.Warn
	case http.StatusRequestEntityTooLarge:
		return logger.Debug
	case http.StatusRequestURITooLong:
		return logger.Debug
	case http.StatusUnsupportedMediaType:
		return logger.Debug
	case http.StatusRequestedRangeNotSatisfiable:
		return logger.Debug
	case http.StatusExpectationFailed:
		return logger.Debug
	case http.StatusTeapot:
		return logger.Debug
	case http.StatusMisdirectedRequest:
		return logger.Debug
	case http.StatusUnprocessableEntity:
		return logger.Debug
	case http.StatusLocked:
		return logger.Debug
	case http.StatusFailedDependency:
		return logger.Debug
	case http.StatusTooEarly:
		return logger.Debug
	case http.StatusUpgradeRequired:
		return logger.Debug
	case http.StatusPreconditionRequired:
		return logger.Debug
	case http.StatusTooManyRequests:
		return logger.Debug
	case http.StatusRequestHeaderFieldsTooLarge:
		return logger.Debug
	case http.StatusUnavailableForLegalReasons:
		return logger.Warn

	case http.StatusInternalServerError:
		return logger.Error
	case http.StatusNotImplemented:
		return logger.Error
	case http.StatusBadGateway:
		return logger.Debug
	case http.StatusServiceUnavailable:
		return logger.Warn
	case http.StatusGatewayTimeout:
		return logger.Warn
	case http.StatusHTTPVersionNotSupported:
		return logger.Debug
	case http.StatusVariantAlsoNegotiates:
		return logger.Debug
	case http.StatusInsufficientStorage:
		return logger.Debug
	case http.StatusLoopDetected:
		return logger.Error
	case http.StatusNotExtended:
		return logger.Debug
	case http.StatusNetworkAuthenticationRequired:
		return logger.Debug
	}
	return logger.Error
}

type cfg struct {
	logger log.Logger
}

type LogOpt = func(cfg) cfg

func WithLogger(logger log.Logger) LogOpt {
	return func(c cfg) cfg {
		c.logger = logger
		return c
	}
}

var logDurationKey = log.DurationKey("http")
var logStartTimeKey = log.StartTimeKey("http")
var logHrefKey = log.HrefKey("http")

func CreateLoggingMiddleware(opts ...LogOpt) func(next http.Handler) http.Handler {
	cfg := cfg{
		logger: log.Get(),
	}
	for _, o := range opts {
		cfg = o(cfg)
	}
	return func(next http.Handler) http.Handler {
		return netHttp.HandlerFunc(func(w netHttp.ResponseWriter, r *netHttp.Request) {
			start := time.Now()
			token := r.Header.Get("Authorization")
			sw := statusWriter{ResponseWriter: w}

			next.ServeHTTP(&sw, r)
			duration := time.Since(start)

			var sub string
			logger := cfg.logger
			if claims, err := jwt.ParseToken(token); err != nil {
				sub = claims.Subject()
				logger.With(log.JWTSubKey, sub)
			}
			logger = logger.With(logDurationKey, log.DurationToMilliseconds(duration), "http.method", r.Method, "http.code", sw.status, logStartTimeKey, start, logHrefKey, r.RequestURI)
			doLog := DefaultCodeToLevel(sw.status, logger)
			doLog("finished unary call with status code ", sw.status)
		})
	}
}
