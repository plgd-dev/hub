package http

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"go.opentelemetry.io/otel/trace"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	err    error
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	if w.status >= 400 && len(b) > 0 && w.err == nil {
		switch w.ResponseWriter.Header().Get(ContentTypeHeaderKey) {
		case ApplicationProtoJsonContentType:
		case message.AppJSON.String():
			var s rpcStatus.Status
			if err := protojson.Unmarshal(b, &s); err == nil {
				w.err = status.ErrorProto(&s)
			}
		}
		if w.err == nil {
			errLen := 1024
			if len(b) < errLen {
				errLen = len(b)
			}
			w.err = fmt.Errorf("%v", string(b[:errLen]))
		}
	}
	n, err := w.ResponseWriter.Write(b)

	return n, err
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	writer, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("not supported by the underlying writer")
	}
	return writer.Hijack()
}

func (w *statusWriter) Flush() {
	f, ok := w.ResponseWriter.(interface{ Flush() })
	if ok {
		f.Flush()
	}
}

var toNil = func(...interface{}) {
	// Do nothing because we don't want to log anything
}

func toDebug(logger log.Logger) func(args ...interface{}) {
	if logger.Check(log.DebugLevel) {
		return logger.Debug
	}
	return nil
}

func toWarn(logger log.Logger) func(args ...interface{}) {
	if logger.Check(log.WarnLevel) {
		return logger.Warn
	}
	return nil
}

func toError(logger log.Logger) func(args ...interface{}) {
	if logger.Check(log.ErrorLevel) {
		return logger.Error
	}
	return nil
}

var defaultCodeToLevel = map[int]func(logger log.Logger) func(args ...interface{}){
	0:                                        toDebug, // websocket returns https status code with 0
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
		v := targerLvl(logger)
		if v == nil {
			return toNil
		}
		return v
	}
	return logger.Error
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func WantToLog(code int, logger log.Logger) bool {
	targerLvl, ok := defaultCodeToLevel[code]
	if ok {
		return targerLvl(logger) != nil
	}
	return true
}

type LogOptions struct {
	logger log.Logger
}

func NewLogOptions() *LogOptions {
	return &LogOptions{
		logger: log.Get(),
	}
}

type LogOpt = func(*LogOptions)

func WithLogger(logger log.Logger) LogOpt {
	return func(c *LogOptions) {
		c.logger = logger
	}
}

var (
	logDurationKey  = log.DurationMSKey
	logStartTimeKey = log.StartTimeKey
)

type jwtMember struct {
	Sub string `json:"sub,omitempty"`
}

type request struct {
	Href   string     `json:"href,omitempty"`
	JWT    *jwtMember `json:"jwt,omitempty"`
	Method string     `json:"method,omitempty"`
}

type response struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

func createLogRequest(r *http.Request) *request {
	bearer := r.Header.Get("Authorization")
	req := request{
		Method: r.Method,
		Href:   r.RequestURI,
	}
	token := strings.SplitN(bearer, " ", 2)
	if len(token) == 2 && strings.ToLower(token[0]) == "bearer" {
		subject, ok := pkgHttpJwt.SubjectFromToken(token[1])
		if !ok {
			return &req
		}

		req.JWT = &jwtMember{
			Sub: subject,
		}
	}
	return &req
}

func CreateLoggingMiddleware(opts ...LogOpt) func(next http.Handler) http.Handler {
	cfg := NewLogOptions()
	for _, o := range opts {
		o(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := statusWriter{ResponseWriter: w}
			next.ServeHTTP(&sw, r)
			duration := time.Since(start)
			logger := cfg.logger
			if !WantToLog(sw.status, logger) {
				return
			}
			req := createLogRequest(r)
			errStr := ""
			if sw.err != nil {
				errStr = sw.err.Error()
			}
			resp := &response{
				Code:  sw.status,
				Error: errStr,
			}
			spanCtx := trace.SpanContextFromContext(r.Context())
			if spanCtx.HasTraceID() {
				logger = logger.With(log.TraceIDKey, spanCtx.TraceID().String())
			}

			logger = logger.With(logDurationKey, log.DurationToMilliseconds(duration), log.RequestKey, req, log.ResponseKey, resp, logStartTimeKey, start, log.ProtocolKey, "HTTP")
			if deadline, ok := r.Context().Deadline(); ok {
				logger = logger.With(log.DeadlineKey, deadline)
			}

			doLog := DefaultCodeToLevel(sw.status, logger)
			doLog("finished unary call with status code ", http.StatusText(sw.status))
		})
	}
}
