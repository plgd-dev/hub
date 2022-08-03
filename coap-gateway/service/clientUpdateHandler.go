package service

import (
	"fmt"

	"github.com/plgd-dev/device/schema/interfaces"
	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const errFmtUpdateResource = "cannot handle update resource: %v: %w"

func getUpdateResourceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtUpdateResource, "", err)
}

// handles resource updates and creation
func clientPostHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	resourceInterface := message.GetResourceInterface(req)
	if resourceInterface == interfaces.OC_IF_CREATE {
		return clientCreateHandler(req, client)
	}
	return clientUpdateHandler(req, client)
}

func clientUpdateHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	_, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getUpdateResourceErr(err))
	}
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getUpdateResourceErr(err))
	}

	code := coapCodes.Changed
	content, err := clientUpdateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Update)
		return nil, statusErrorf(code, errFmtUpdateResource, fmt.Sprintf(" /%v%v", deviceID, href), err)
	}
	if content == nil || len(content.Data) == 0 {
		return client.createResponse(code, req.Token, coapMessage.TextPlain, nil), nil
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		return nil, statusErrorf(code, "cannot encode response for update resource /%v%v: %w", deviceID, href, err)
	}
	return client.createResponse(code, req.Token, mediaType, content.Data), nil
}

func clientUpdateDeviceHandler(req *mux.Message, client *Client, deviceID, href string) (*commands.Content, error) {
	updateCommand, err := coapconv.NewUpdateResourceRequest(commands.NewResourceID(deviceID, href), req, client.RemoteAddr().String())
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
