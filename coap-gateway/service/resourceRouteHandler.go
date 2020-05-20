package service

import (
	"fmt"

	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
)

var resourceRoute = "oic/route"

func resourceRouteHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.POST:
		clientUpdateHandler(s, req, client)
	case coapCodes.GET:
		req.Options.Observe
		if observe, err := req.Options.Observe(); err == nil {
			clientObserveHandler(s, req, client, observe)
			return
		}
		clientRetrieveHandler(s, req, client)
	default:
		deviceID := getDeviceID(client)
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v, Href %v: unsupported method %v", deviceID, req.Msg.PathString(), req.Msg.Code()), s, client, coapCodes.MethodNotAllowed)
	}
}
