package service

import (
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/log"
)

func logAndWriteErrorResponse(err error, s mux.ResponseWriter, client *Client, code codes.Code) {
	msg := s.NewResponse(code)
	if err != nil {
		log.Errorf("%v", err)
	}
	if msg != nil {
		if client != nil && client.server.SendErrorTextInResponse {
			msg.SetOption(message.ContentFormat, message.TextPlain)
			msg.SetPayload([]byte(err.Error()))
		} else {
			msg.SetOption(message.ContentFormat, message.AppCBOR)
			msg.SetPayload([]byte{0xA0}) // empty object
		}
		err = s.WriteMsg(msg)
		if err != nil {
			log.Errorf("cannot send error to %v: %v", getDeviceId(client), err)
		}
		decodeMsgToDebug(client, msg, "SEND-ERROR")
	}
}
