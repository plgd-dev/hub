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

func (w *statusWriter) Flush() {
	f, ok := w.ResponseWriter.(interface{ Flush() })
	if ok {
		f.Flush()
	}
}

func toDebug(logger log.Logger) func(args ...interface{}) {
	return logger.Debug
}

func toWarn(logger log.Logger) func(args ...interface{}) {
	return logger.Warn
}

func toError(logger log.Logger) func(args ...interface{}) {
	return logger.Error
}

var defaultCodeToLevel = map[int]func(logger log.Logger) func(args ...interface{}){
	http.StatusContinue:                      toDebug,
	http.StatusSwitchingProtocols:            toDebug,
	http.StatusProcessing:                    toDebug,
	http.StatusEarlyHints:                    toDebug,
	http.StatusOK:                            toDebug,
	http.StatusCreated:                       toDebug,
	http.StatusAccepted:                      toDebug,
	http.StatusNonAuthoritativeInfo:          toDebug,
	http.StatusNoContent:                     toDebug,
	http.StatusResetContent:                  toDebug,
	http.StatusPartialContent:                toDebug,
	http.StatusMultiStatus:                   toDebug,
	http.StatusAlreadyReported:               toDebug,
	http.StatusIMUsed:                        toDebug,
	http.StatusMultipleChoices:               toDebug,
	http.StatusMovedPermanently:              toDebug,
	http.StatusFound:                         toDebug,
	http.StatusSeeOther:                      toDebug,
	http.StatusNotModified:                   toDebug,
	http.StatusUseProxy:                      toDebug,
	http.StatusTemporaryRedirect:             toDebug,
	http.StatusPermanentRedirect:             toDebug,
	http.StatusBadRequest:                    toDebug,
	http.StatusUnauthorized:                  toDebug,
	http.StatusPaymentRequired:               toDebug,
	http.StatusForbidden:                     toWarn,
	http.StatusNotFound:                      toDebug,
	http.StatusMethodNotAllowed:              toDebug,
	http.StatusNotAcceptable:                 toDebug,
	http.StatusProxyAuthRequired:             toDebug,
	http.StatusRequestTimeout:                toDebug,
	http.StatusConflict:                      toDebug,
	http.StatusGone:                          toDebug,
	http.StatusLengthRequired:                toDebug,
	http.StatusPreconditionFailed:            toWarn,
	http.StatusRequestEntityTooLarge:         toDebug,
	http.StatusRequestURITooLong:             toDebug,
	http.StatusUnsupportedMediaType:          toDebug,
	http.StatusRequestedRangeNotSatisfiable:  toDebug,
	http.StatusExpectationFailed:             toDebug,
	http.StatusTeapot:                        toDebug,
	http.StatusMisdirectedRequest:            toDebug,
	http.StatusUnprocessableEntity:           toDebug,
	http.StatusLocked:                        toDebug,
	http.StatusFailedDependency:              toDebug,
	http.StatusTooEarly:                      toDebug,
	http.StatusUpgradeRequired:               toDebug,
	http.StatusPreconditionRequired:          toDebug,
	http.StatusTooManyRequests:               toDebug,
	http.StatusRequestHeaderFieldsTooLarge:   toDebug,
	http.StatusUnavailableForLegalReasons:    toWarn,
	http.StatusInternalServerError:           toError,
	http.StatusNotImplemented:                toError,
	http.StatusBadGateway:                    toDebug,
	http.StatusServiceUnavailable:            toWarn,
	http.StatusGatewayTimeout:                toWarn,
	http.StatusHTTPVersionNotSupported:       toDebug,
	http.StatusVariantAlsoNegotiates:         toDebug,
	http.StatusInsufficientStorage:           toDebug,
	http.StatusLoopDetected:                  toError,
	http.StatusNotExtended:                   toDebug,
	http.StatusNetworkAuthenticationRequired: toDebug,
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func DefaultCodeToLevel(code int, logger log.Logger) func(args ...interface{}) {
	targerLvl, ok := defaultCodeToLevel[code]
	if ok {
		return targerLvl(logger)
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
			if claims, err := jwt.ParseToken(token); err == nil {
				sub = claims.Subject()
				logger.With(log.JWTSubKey, sub)
			}
			logger = logger.With(logDurationKey, log.DurationToMilliseconds(duration), "http.method", r.Method, "http.code", sw.status, logStartTimeKey, start, logHrefKey, r.RequestURI)
			doLog := DefaultCodeToLevel(sw.status, logger)
			doLog("finished unary call with status code ", sw.status)
		})
	}
}
