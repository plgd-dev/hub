package service

import (
	"time"

	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

const logRequestKey = "coap.request"
const logRequestDeadlineKey = "coap.request.deadline"
const logResponseKey = "coap.response"
const logRequestToClientKey = "request to client"

const logNotificationKey = "coap.notification"

var logCorrelationIDKey = log.CorrelationIDKey
var logDeviceIDKey = log.DeviceIDKey
var logDurationKey = log.DurationKey("coap")
var logServiceKey = log.ServiceKey("coap")
var logPathKey = log.HrefKey("coap")
var logStartTimeKey = log.StartTimeKey("coap")

func logToLevel(respCode codes.Code, logger log.Logger) func(args ...interface{}) {
	switch respCode {
	case codes.Created, codes.Deleted, codes.Valid, codes.Changed, codes.Continue, codes.Content, codes.NotFound:
		return logger.Debug
	case codes.GatewayTimeout, codes.Forbidden, codes.Abort, codes.ServiceUnavailable, codes.PreconditionFailed, codes.MethodNotAllowed, codes.NotAcceptable, codes.BadOption, codes.Unauthorized, codes.BadRequest, codes.RequestEntityIncomplete, codes.RequestEntityTooLarge, codes.UnsupportedMediaType:
		return logger.Warn
	case codes.NotImplemented, codes.InternalServerError, codes.BadGateway, codes.ProxyingNotSupported:
		return logger.Error
	default:
		return logger.Error
	}
}

func (client *Client) getLogger() log.Logger {
	logger := client.server.logger
	if deviceID := client.deviceID(); deviceID != "" {
		logger = logger.With(logDeviceIDKey, deviceID)
	}
	if v, err := client.GetAuthorizationContext(); err == nil && v.GetJWTClaims().Subject() != "" {
		logger = logger.With(log.JWTSubKey, v.GetJWTClaims().Subject())
	}
	return logger
}

func (client *Client) Errorf(fmt string, args ...interface{}) {
	client.getLogger().Errorf(fmt, args...)
}

func (client *Client) Debugf(fmt string, args ...interface{}) {
	client.getLogger().Debugf(fmt, args...)
}

func (client *Client) Infof(fmt string, args ...interface{}) {
	client.getLogger().Infof(fmt, args...)
}

func (client *Client) logWithMuxRequestResponse(req *mux.Message, resp *pool.Message) log.Logger {
	var rq *pool.Message
	if req != nil {
		tmp, err := client.server.messagePool.ConvertFrom(req.Message)
		if err == nil {
			rq = tmp
		}
	}
	return client.logWithRequestResponse(rq, resp)
}

func (client *Client) logWithRequestResponse(req *pool.Message, resp *pool.Message) log.Logger {
	logger := client.getLogger()
	if req != nil {
		startTime, ok := req.Context().Value(&logStartTimeKey).(time.Time)
		if ok {
			logger = logger.With(logStartTimeKey, startTime, logDurationKey, log.DurationToMilliseconds(time.Since(startTime)))
		}
		deadline, ok := req.Context().Deadline()
		if ok {
			logger = logger.With(logRequestDeadlineKey, deadline)
		}
		logger = client.reqToLogger(req, logger, resp == nil)
	}
	if resp != nil {
		rsp := coapgwMessage.ToJson(resp, client.server.config.Log.DumpBody, req == nil)
		logger = logger.With(logResponseKey, rsp)
	}
	return logger
}

func (client *Client) logRequest(msg string, req *pool.Message, resp *pool.Message) {
	logger := client.logWithRequestResponse(req, resp)
	if resp != nil {
		logToLevel(resp.Code(), logger)(msg)
		return
	}
	logger.Debug(msg)
}

func (client *Client) reqToLogger(req *pool.Message, logger log.Logger, withToken bool) log.Logger {
	rq := coapgwMessage.ToJson(req, client.server.config.Log.DumpBody, withToken)
	if rq.Path != "" {
		logger = logger.With(logPathKey, rq.Path)
		rq.Path = ""
	}
	return logger.With(logRequestKey, rq)
}

func (client *Client) ErrorfRequest(req *mux.Message, fmt string, args ...interface{}) {
	client.logWithMuxRequestResponse(req, nil).Errorf(fmt, args...)
}

func (client *Client) logClientRequest(req *mux.Message, resp *pool.Message) {
	var rq *pool.Message
	if req != nil {
		tmp, err := client.server.messagePool.ConvertFrom(req.Message)
		if err == nil {
			rq = tmp
		}
	}
	client.logRequest("request from client", rq, resp)
}

func (client *Client) logNotification(logMsg, path string, notification *pool.Message) {
	logger := client.getLogger()
	if path != "" {
		logger = logger.With(logPathKey, path)
	}
	if notification != nil {
		rsp := coapgwMessage.ToJson(notification, client.server.config.Log.DumpBody, true)
		logger = logger.With(logNotificationKey, rsp)
	}
	logToLevel(notification.Code(), logger)(logMsg)
}

func (client *Client) logNotificationToClient(path string, notification *pool.Message) {
	client.logNotification("notification to client", path, notification)
}

func (client *Client) logNotificationFromClient(path string, notification *pool.Message) {
	client.logNotification("notification from client", path, notification)
}
