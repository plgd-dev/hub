package service

import (
	"context"

	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

func NewMessageWithCode(code codes.Code) *pool.Message {
	msg := pool.NewMessage(context.Background())
	msg.SetCode(code)
	return msg
}
