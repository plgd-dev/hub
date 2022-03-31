package service

import (
	"fmt"

	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

func clientCreateHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(req, fmt.Errorf("DeviceId: %v: cannot handle create resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(req, fmt.Errorf("DeviceId: %v: cannot handle create resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	code := coapCodes.Created
	content, err := clientCreateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Create)
		client.logAndWriteErrorResponse(req, fmt.Errorf("DeviceId: %v: cannot handle create resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		client.sendResponse(req, code, req.Token, coapMessage.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(req, fmt.Errorf("DeviceId: %v: cannot encode response for create resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}
	client.sendResponse(req, code, req.Token, mediaType, content.Data)
}

func clientCreateDeviceHandler(req *mux.Message, client *Client, deviceID, href string) (*commands.Content, error) {
	createCommand, err := coapconv.NewCreateResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, err
	}

	createdEvent, err := client.server.raClient.SyncCreateResource(req.Context, "*", createCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(createdEvent)
	if err != nil {
		return nil, err
	}

	return content, nil
}