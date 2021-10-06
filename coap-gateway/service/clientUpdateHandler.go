package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/pkg/ocf"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
)

//handles resource updates and creation
func clientPostHandler(req *mux.Message, client *Client) {
	resourceInterface := getResourceInterface(req)
	if resourceInterface == ocf.OC_IF_CREATE {
		clientCreateHandler(req, client)
		return
	}
	clientUpdateHandler(req, client)
}

func clientUpdateHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	code := coapCodes.Changed
	content, err := clientUpdateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Update)
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for update resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientUpdateDeviceHandler(req *mux.Message, client *Client, deviceID, href string) (*commands.Content, error) {
	updateCommand, err := coapconv.NewUpdateResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, err
	}

	updatedEvent, err := client.server.raClient.SyncUpdateResource(req.Context, "*", updateCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(updatedEvent)
	if err != nil {
		return nil, err
	}

	return content, nil
}
