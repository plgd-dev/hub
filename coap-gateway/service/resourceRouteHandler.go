package service

import (
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

func resourceRouteHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	switch req.Code {
	case coapCodes.POST:
		//handles resource updates and creation
		return clientPostHandler(req, client)
	case coapCodes.DELETE:
		return clientDeleteHandler(req, client)
	case coapCodes.GET:
		if observe, err := req.Options.Observe(); err == nil {
			return clientObserveHandler(req, client, observe)
		}
		return clientRetrieveHandler(req, client)
	default:
		path, _ := req.Options.Path()
		return nil, statusErrorf(coapCodes.NotFound, "unknown path %v", path)
	}
}
