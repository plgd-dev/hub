package service

import (
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
)

func clientDeleteHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle delete resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle delete resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	code := coapCodes.Deleted
	content, err := clientDeleteResourceHandler(req, client, deviceID, href, authCtx.GetUserID())
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Delete)
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot delete resource /%v%v from device: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}

	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot delete resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientDeleteResourceHandler(req *mux.Message, client *Client, deviceID, href, userID string) (*commands.Content, error) {
	deleteCommand, err := coapconv.NewDeleteResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, err
	}

	deletedCommand, err := client.server.raClient.SyncDeleteResource(req.Context, deleteCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(deletedCommand)
	if err != nil {
		return nil, err
	}
	return content, nil
}
