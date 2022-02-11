package service

import (
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

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

func (client *Client) logDeviceRequest(req *mux.Message, resp *pool.Message) {
	logger := client.getLogger()
	if req != nil {
		tmp, err := client.server.messagePool.ConvertFrom(req.Message)
		if err == nil {
			rq := coapgwMessage.ToJson(tmp, client.server.config.Log.DumpBody)
			logger = logger.With("req", rq)
		}
	}
	if resp != nil {
		rsp := coapgwMessage.ToJson(resp, client.server.config.Log.DumpBody)
		logger = logger.With("resp", rsp)
	}
	logger.Info("client request")
}

func (client *Client) logNotificationFromDevice(path string, notification *pool.Message) {
	logger := client.getLogger()
	if path != "" {
		logger = logger.With("path", path)
	}
	if notification != nil {
		rsp := coapgwMessage.ToJson(notification, client.server.config.Log.DumpBody)
		logger = logger.With("notification", rsp)
	}
	logger.Info("notification from client")
}
