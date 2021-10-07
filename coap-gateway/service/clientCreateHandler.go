package service

import (
	"fmt"

	"github.com/plgd-dev/hub/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
)

func clientCreateHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle create resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle create resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	code := coapCodes.Created
	content, err := clientCreateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Create)
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle create resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for create resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
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
