package service

import (
	"fmt"

	coapMessage "github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const errFmtCreateResource = "cannot handle create resource%v: %w"

func getCreateResourceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtCreateResource, "", err)
}

func clientCreateHandler(req *mux.Message, client *session) (*pool.Message, error) {
	_, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getCreateResourceErr(err))
	}
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getCreateResourceErr(err))
	}

	code := coapCodes.Created
	content, err := clientCreateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		code = coapconv.GrpcErr2CoapCode(err, coapconv.Create)
		return nil, statusErrorf(code, errFmtCreateResource, fmt.Sprintf(" /%v%v", deviceID, href), err)
	}
	if len(content.GetData()) == 0 {
		return client.createResponse(code, req.Token(), coapMessage.TextPlain, nil), nil
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.GetContentType())
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "cannot encode response for create resource %v: %w", fmt.Sprintf(" /%v%v", deviceID, href), err)
	}
	return client.createResponse(code, req.Token(), mediaType, content.GetData()), nil
}

func clientCreateDeviceHandler(req *mux.Message, client *session, deviceID, href string) (*commands.Content, error) {
	createCommand, err := coapconv.NewCreateResourceRequest(commands.NewResourceID(deviceID, href), req, client.RemoteAddr().String())
	if err != nil {
		return nil, err
	}

	createdEvent, err := client.server.raClient.SyncCreateResource(req.Context(), "*", createCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(createdEvent)
	if err != nil {
		return nil, err
	}

	return content, nil
}
