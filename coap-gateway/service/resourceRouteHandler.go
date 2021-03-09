package service

import (
	"fmt"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
)

func resourceRouteHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		clientPostHandler(req, client)
	case coapCodes.DELETE:
		clientDeleteHandler(req, client)
	case coapCodes.GET:
		if observe, err := req.Options.Observe(); err == nil {
			clientObserveHandler(req, client, observe)
			return
		}
		clientRetrieveHandler(req, client)
	default:
		deviceID := getDeviceID(client)
		path, _ := req.Options.Path()
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v, Href %v: unsupported method %v", deviceID, path, req.Code), coapCodes.MethodNotAllowed, req.Token)
	}
}
