package service

import (
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

const logRequestKey = "req"
const logResponseKey = "resp"

func (client *Client) getLogger() log.Logger {
	logger := log.Get()
	if deviceID := client.deviceID(); deviceID != "" {
		logger = logger.With("deviceId", deviceID)
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

func (client *Client) logRequest(msg string, req *pool.Message, resp *pool.Message) {
	logger := client.getLogger()
	if req != nil {
		rq := coapgwMessage.ToJson(req, client.server.config.Log.DumpBody, resp == nil)
		logger = logger.With(logRequestKey, rq)
	}
	if resp != nil {
		rsp := coapgwMessage.ToJson(resp, client.server.config.Log.DumpBody, req == nil)
		logger = logger.With(logResponseKey, rsp)
	}
	logger.Info(msg)
}

func (client *Client) ErrorfRequest(req *mux.Message, fmt string, args ...interface{}) {
	logger := client.getLogger()
	if req != nil {
		tmp, err := client.server.messagePool.ConvertFrom(req.Message)
		if err == nil {
			rq := coapgwMessage.ToJson(tmp, client.server.config.Log.DumpBody, true)
			logger = logger.With(logRequestKey, rq)
		}
	}
	logger.Errorf(fmt, args...)
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

func (client *Client) logServiceRequest(req *pool.Message, resp *pool.Message) {
	client.logRequest("request to client", req, resp)
}

func (client *Client) logNotification(logMsg, path string, notification *pool.Message) {
	logger := client.getLogger()
	if path != "" {
		logger = logger.With("path", path)
	}
	if notification != nil {
		rsp := coapgwMessage.ToJson(notification, client.server.config.Log.DumpBody, true)
		logger = logger.With("notification", rsp)
	}
	logger.Info(logMsg)
}

func (client *Client) logNotificationToClient(path string, notification *pool.Message) {
	client.logNotification("notification to client", path, notification)
}

func (client *Client) logNotificationFromClient(path string, notification *pool.Message) {
	client.logNotification("notification from client", path, notification)
}
