package service

import (
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
)

var resourceRoute = "oic/route"

func resourceRouteHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.POST:
		clientUpdateHandler(s, req, client)
	case coapCodes.GET:
		if observe, ok := req.Msg.Option(gocoap.Observe).(uint32); ok {
			clientObserveHandler(s, req, client, observe)
			return
		}
		clientRetrieveHandler(s, req, client)
	default:
		deviceId := getDeviceId(client)
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v, Href %v: unsupported method %v", deviceId, req.Msg.PathString(), req.Msg.Code()), s, client, coapCodes.MethodNotAllowed)
	}
}
