package service

import (
	"fmt"

	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
)

var resourceRoute = "oic/route"

func resourceRouteHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		clientUpdateHandler(s, req, client)
	case coapCodes.GET:
		if observe, err := req.Options.Observe(); err == nil {
			clientObserveHandler(s, req, client, observe)
			return
		}
		clientRetrieveHandler(s, req, client)
	default:
		deviceID := getDeviceID(client)
		path, _ := req.Options.Path()
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v, Href %v: unsupported method %v", deviceID, path, req.Code),  coapCodes.MethodNotAllowed, req.Token)
	}
}
