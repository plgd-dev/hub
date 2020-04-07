package service

import (
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/log"
)

func logAndWriteErrorResponse(err error, s gocoap.ResponseWriter, client *Client, code coapCodes.Code) {
	msg := s.NewResponse(code)
	if err != nil {
		log.Errorf("%v", err)
	}
	if msg != nil {
		if client != nil && client.server.SendErrorTextInResponse {
			msg.SetOption(gocoap.ContentFormat, gocoap.TextPlain)
			msg.SetPayload([]byte(err.Error()))
		} else {
			msg.SetOption(gocoap.ContentFormat, gocoap.AppCBOR)
			msg.SetPayload([]byte{0xA0}) // empty object
		}
		err = s.WriteMsg(msg)
		if err != nil {
			log.Errorf("cannot send error to %v: %v", getDeviceId(client), err)
		}
		decodeMsgToDebug(client, msg, "SEND-ERROR")
	}
}
