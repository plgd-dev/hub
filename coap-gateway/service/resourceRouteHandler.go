package service

import (
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

func resourceRouteHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		// handles resource updates and creation
		return clientPostHandler(req, client)
	case coapCodes.DELETE:
		return clientDeleteHandler(req, client)
	case coapCodes.GET:
		if observe, err := req.Options().Observe(); err == nil {
			return clientObserveHandler(req, client, observe)
		}
		return clientRetrieveHandler(req, client)
	default:
		path, _ := req.Options().Path()
		return nil, statusErrorf(coapCodes.NotFound, "unknown path %v", path)
	}
}
