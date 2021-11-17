package service

import (
	"fmt"

	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/coap-gateway/service/message"
	"github.com/plgd-dev/hub/pkg/log"
)

func resourceObserveHandler(req *mux.Message, client *Client, observe uint32) {
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %w", client.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}
	log.Debugf("observe(%v) /%v%v resource\n", observe, deviceID, href)
	if err := client.handler.ObserveResource(deviceID, href, observe); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %w", client.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
	}
}

func resourceRetrieveHandler(req *mux.Message, client *Client) {
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle retrieve resource: %w", client.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}
	log.Debugf("retrieve /%v%v resource\n", deviceID, href)
	if err := client.handler.RetrieveResource(deviceID, href); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle retrieve resource: %w", client.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
	}
	client.sendResponse(coapCodes.Content, req.Token, coapMessage.TextPlain, nil)
}

func resourceHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.GET:
		if observe, err := req.Options.Observe(); err == nil {
			resourceObserveHandler(req, client, observe)
			return
		}
		resourceRetrieveHandler(req, client)
	default:
		path, _ := req.Options.Path()
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v, Href %v: unsupported method %v", client.GetDeviceID(), path, req.Code), coapCodes.MethodNotAllowed, req.Token)
	}
}
