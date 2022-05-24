package service

import (
	"fmt"

	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const errFmtDeleteResource = "cannot handle delete resource%v: %w"

func getDeleteResourceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtDeleteResource, "", err)
}

func clientDeleteHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getDeleteResourceErr(err))
	}
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getDeleteResourceErr(err))
	}

	code := coapCodes.Deleted
	content, err := clientDeleteResourceHandler(req, client, deviceID, href, authCtx.GetUserID())
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Delete)
		return nil, statusErrorf(code, errFmtDeleteResource, fmt.Sprintf(" /%v%v", deviceID, href), err)
	}

	if content == nil || len(content.Data) == 0 {
		return client.createResponse(code, req.Token, coapMessage.TextPlain, nil), nil
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		return nil, statusErrorf(code, errFmtDeleteResource, fmt.Sprintf(" /%v%v", deviceID, href), err)
	}
	return client.createResponse(code, req.Token, mediaType, content.Data), nil
}

func clientDeleteResourceHandler(req *mux.Message, client *Client, deviceID, href, userID string) (*commands.Content, error) {
	deleteCommand, err := coapconv.NewDeleteResourceRequest(commands.NewResourceID(deviceID, href), req, client.RemoteAddr().String())
	if err != nil {
		return nil, err
	}

	deletedCommand, err := client.server.raClient.SyncDeleteResource(req.Context, "*", deleteCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(deletedCommand)
	if err != nil {
		return nil, err
	}
	return content, nil
}
