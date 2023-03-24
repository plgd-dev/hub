package service

import (
	"time"

	"github.com/plgd-dev/device/v2/schema/plgdtime"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

func plgdTimeGetHandler(req *mux.Message, client *session) (*pool.Message, error) {
	resp := plgdtime.PlgdTimeUpdate{
		Time: time.Now().Format(time.RFC3339Nano),
	}
	accept, out, err := encodeResponse(resp, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "cannot encode plgdTime response: %w", err)
	}
	return client.createResponse(coapCodes.Content, req.Token(), accept, out), nil
}

func plgdTimeHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.GET:
		return plgdTimeGetHandler(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code())
	}
}
