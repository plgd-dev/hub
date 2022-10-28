package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

const logNotificationKey = "notification"

var toNil = func(args ...interface{}) {
	// Do nothing because we don't want to log anything
}

func toDebug(logger log.Logger) func(args ...interface{}) {
	if logger.Check(zapcore.DebugLevel) {
		return logger.Debug
	}
	return nil
}

func toWarn(logger log.Logger) func(args ...interface{}) {
	if logger.Check(zapcore.WarnLevel) {
		return logger.Warn
	}
	return nil
}

func toError(logger log.Logger) func(args ...interface{}) {
	if logger.Check(zapcore.ErrorLevel) {
		return logger.Error
	}
	return nil
}

var defaultCodeToLevel = map[codes.Code]func(logger log.Logger) func(args ...interface{}){
	codes.Created:  toDebug,
	codes.Deleted:  toDebug,
	codes.Valid:    toDebug,
	codes.Changed:  toDebug,
	codes.Continue: toDebug,
	codes.Content:  toDebug,
	codes.NotFound: toDebug,

	codes.GatewayTimeout:          toWarn,
	codes.Forbidden:               toWarn,
	codes.Abort:                   toWarn,
	codes.ServiceUnavailable:      toWarn,
	codes.PreconditionFailed:      toWarn,
	codes.MethodNotAllowed:        toWarn,
	codes.NotAcceptable:           toWarn,
	codes.BadOption:               toWarn,
	codes.Unauthorized:            toWarn,
	codes.BadRequest:              toWarn,
	codes.RequestEntityIncomplete: toWarn,
	codes.RequestEntityTooLarge:   toWarn,
	codes.UnsupportedMediaType:    toWarn,

	codes.NotImplemented:       toError,
	codes.InternalServerError:  toError,
	codes.BadGateway:           toError,
	codes.ProxyingNotSupported: toError,
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func DefaultCodeToLevel(code codes.Code, logger log.Logger) func(args ...interface{}) {
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

func WantToLog(code codes.Code, logger log.Logger) bool {
	targerLvl, ok := defaultCodeToLevel[code]
	if ok {
		return targerLvl(logger) != nil
	}
	return true
}

func (c *session) getLogger() log.Logger {
	logger := c.server.logger
	deviceID := c.deviceID()
	if deviceID != "" {
		logger = logger.With(log.DeviceIDKey, deviceID)
	}
	select {
	case <-c.coapConn.Done():
		logger = logger.With("closedConnection", true)
	default:
		if c.coapConn.Context().Err() != nil {
			logger = logger.With("closingConnection", true)
		}
	}
	return logger
}

func (c *session) Errorf(fmt string, args ...interface{}) {
	logger := c.getLogger()
	logger.Errorf(fmt, args...)
}

func (c *session) Debugf(fmt string, args ...interface{}) {
	logger := c.getLogger()
	logger.Debugf(fmt, args...)
}

func (c *session) Infof(fmt string, args ...interface{}) {
	logger := c.getLogger()
	logger.Infof(fmt, args...)
}

type jwtMember struct {
	Sub string `json:"sub,omitempty"`
}

type logCoapMessage struct {
	JWT    *jwtMember `json:"jwt,omitempty"`
	Method string     `json:"method,omitempty"`
	coapgwMessage.JsonCoapMessage
}

func (c *session) loggerWithRequestResponse(logger log.Logger, req *pool.Message, resp *pool.Message) log.Logger {
	var spanCtx trace.SpanContext
	if req != nil {
		spanCtx = trace.SpanContextFromContext(req.Context())
		startTime, ok := req.Context().Value(&log.StartTimeKey).(time.Time)
		if ok {
			logger = logger.With(log.StartTimeKey, startTime, log.DurationMSKey, log.DurationToMilliseconds(time.Since(startTime)))
		}
		deadline, ok := req.Context().Deadline()
		if ok {
			logger = logger.With(log.DeadlineKey, deadline)
		}
		logMsg := c.msgToLogCoapMessage(req, resp == nil)
		logger = logger.With(log.RequestKey, logMsg)
	}
	if resp != nil {
		if !spanCtx.IsValid() {
			spanCtx = trace.SpanContextFromContext(resp.Context())
		}
		logMsg := c.msgToLogCoapMessage(resp, req == nil)
		if req != nil {
			logMsg.JWT = nil
		}
		logger = logger.With(log.ResponseKey, logMsg)
	}
	if spanCtx.HasTraceID() {
		logger = logger.With(log.TraceIDKey, spanCtx.TraceID().String())
	}
	return logger.With(log.ProtocolKey, "COAP")
}

func (c *session) logRequestResponse(req *mux.Message, resp *pool.Message, err error) {
	logger := c.getLogger()
	if resp != nil && !WantToLog(resp.Code(), logger) {
		return
	}
	logger = c.loggerWithRequestResponse(c.getLogger(), req.Message, resp)
	if err != nil {
		logger = logger.With(log.ErrorKey, err.Error())
	}
	if resp != nil {
		msg := fmt.Sprintf("finished unary call from the device with code %v", resp.Code())
		DefaultCodeToLevel(resp.Code(), logger)(msg)
		return
	}
	logger.Debug("finished unary call from the device")
}

func (c *session) msgToLogCoapMessage(req *pool.Message, withToken bool) logCoapMessage {
	rq := coapgwMessage.ToJson(req, c.server.config.Log.DumpBody, withToken)
	var sub string
	if v, err := c.GetAuthorizationContext(); err == nil {
		sub = v.GetJWTClaims().Subject()
	}
	dumpReq := logCoapMessage{
		JsonCoapMessage: rq,
	}
	if sub != "" {
		dumpReq.JWT = &jwtMember{
			Sub: sub,
		}
	}
	if req.Code() >= codes.GET && req.Code() <= codes.DELETE {
		dumpReq.Method = rq.Code
		dumpReq.Code = ""
	}
	return dumpReq
}

func (c *session) logNotification(logMsg, path string, notification *pool.Message) {
	logger := c.getLogger()
	if notification != nil && !WantToLog(notification.Code(), logger) {
		return
	}
	if notification != nil {
		rsp := c.msgToLogCoapMessage(notification, true)
		rsp.Path = path
		logger = logger.With(logNotificationKey, rsp)
		spanCtx := trace.SpanContextFromContext(notification.Context())
		if spanCtx.HasTraceID() {
			logger = logger.With(log.TraceIDKey, spanCtx.TraceID().String())
		}
	}
	DefaultCodeToLevel(notification.Code(), logger.With(log.ProtocolKey, "COAP"))(logMsg)
}

const logNotificationDefaultCode = "unknown"

func (c *session) logNotificationToClient(path string, notification *pool.Message) {
	code := logNotificationDefaultCode
	if notification != nil {
		code = notification.Code().String()
	}
	c.logNotification(fmt.Sprintf("notification to the device was send with code %v", code), path, notification)
}

func (c *session) logNotificationFromClient(path string, notification *pool.Message) {
	code := logNotificationDefaultCode
	if notification != nil {
		code = notification.Code().String()
	}
	c.logNotification(fmt.Sprintf("notification from the device was received with code %v", code), path, notification)
}
