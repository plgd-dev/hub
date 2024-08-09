package service

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

const (
	certificateCommonName = "certificateCommonName"
	enrollmentGroup       = "enrollmentGroup"
	errorKey              = "error"
	remoterAddr           = "remoteAddress"
	localEndpointsKey     = "localEndpoints"
)

type logCoapMessage struct {
	Method string `json:"method,omitempty"`
	coapgwMessage.JsonCoapMessage
}

func msgToLogCoapMessage(req *pool.Message, withBody bool) logCoapMessage {
	rq := coapgwMessage.ToJson(req, withBody, false)
	dumpReq := logCoapMessage{
		JsonCoapMessage: rq,
	}
	if req.Code() >= codes.GET && req.Code() <= codes.DELETE {
		dumpReq.Method = rq.Code
		dumpReq.Code = ""
	}
	return dumpReq
}

func (s *Session) getLogger() log.Logger {
	logger := s.server.logger.With(remoterAddr, s.coapConn.RemoteAddr().String())
	if cn := s.String(); cn != "" {
		logger = logger.With(certificateCommonName, cn)
	}
	if s.enrollmentGroup != nil {
		logger = logger.With(enrollmentGroup, s.enrollmentGroup.GetId())
	}
	return logger
}

func (s *Session) loggerWithReqResp(ctx context.Context, logger log.Logger, startTime time.Time, req *mux.Message, resp *pool.Message, err error) log.Logger {
	logger = logger.With(log.StartTimeKey, startTime, log.DurationMSKey, log.DurationToMilliseconds(time.Since(startTime)))
	if err != nil {
		logger = logger.With(errorKey, err.Error())
	}
	deadline, ok := ctx.Deadline()
	if ok {
		logger = logger.With(log.DeadlineKey, deadline)
	}
	if req != nil {
		logger = logger.With(log.RequestKey, msgToLogCoapMessage(req.Message, s.server.config.Log.DumpBody))
	}
	if resp != nil {
		logger = logger.With(log.ResponseKey, msgToLogCoapMessage(resp, s.server.config.Log.DumpBody))
	}
	return logger.With(log.ProtocolKey, "COAP")
}

func (s *Session) logRequestResponse(ctx context.Context, startTime time.Time, req *mux.Message, resp *pool.Message, err error) {
	logger := s.getLogger()
	if resp != nil && !coapgwService.WantToLog(resp.Code(), logger) {
		return
	}
	logger = s.loggerWithReqResp(ctx, logger, startTime, req, resp, err)
	if resp != nil {
		msg := fmt.Sprintf("finished unary call from the device with code %v", resp.Code())
		coapgwService.DefaultCodeToLevel(resp.Code(), logger)(msg)
		return
	}
	logger.Debug("finished unary call from the device")
}
