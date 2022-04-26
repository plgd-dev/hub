package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const errFmtRetrieveResource = "cannot handle retrieve resource %v: %w"

func getRetrieveResourceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtRetrieveResource, "", err)
}

func clientRetrieveHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	_, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getRetrieveResourceErr(err))
	}

	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getRetrieveResourceErr(err))
	}

	var content *commands.Content
	var code coapCodes.Code
	resourceInterface := message.GetResourceInterface(req)
	if resourceInterface == "" {
		content, code, err = clientRetrieveFromResourceShadowHandler(req.Context, client, deviceID, href)
		if err != nil {
			return nil, statusErrorf(code, errFmtRetrieveResource, fmt.Sprintf(" /%v%v from resource shadow", deviceID, href), err)
		}
	} else {
		code = coapCodes.Content
		content, err = clientRetrieveFromDeviceHandler(req, client, deviceID, href)
		if err != nil {
			code = coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve)
			return nil, statusErrorf(code, errFmtRetrieveResource, fmt.Sprintf(" /%v%v from device", deviceID, href), err)
		}
	}

	if content == nil || len(content.Data) == 0 {
		return client.createResponse(code, req.Token, coapMessage.TextPlain, nil), nil
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		return nil, statusErrorf(code, errFmtRetrieveResource, fmt.Sprintf(" /%v%v", deviceID, href), err)
	}
	return client.createResponse(code, req.Token, mediaType, content.Data), nil
}

func clientRetrieveFromResourceShadowHandler(ctx context.Context, client *Client, deviceID, href string) (*commands.Content, coapCodes.Code, error) {
	RetrieveResourcesClient, err := client.server.rdClient.GetResources(ctx, &pbGRPC.GetResourcesRequest{
		ResourceIdFilter: []string{
			commands.NewResourceID(deviceID, href).ToString(),
		},
	})
	if err != nil {
		return nil, coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve), err
	}
	defer func() {
		if err := RetrieveResourcesClient.CloseSend(); err != nil {
			client.Errorf("failed to close retrieve devices client: %w", err)
		}
	}()
	for {
		resourceValue, err := RetrieveResourcesClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve), err
		}
		if resourceValue.GetData().GetResourceId().GetDeviceId() == deviceID && resourceValue.GetData().GetResourceId().GetHref() == href && resourceValue.GetData().Content != nil {
			return resourceValue.GetData().Content, coapCodes.Content, nil
		}
	}
	return nil, coapCodes.NotFound, fmt.Errorf("not found")
}

func clientRetrieveFromDeviceHandler(req *mux.Message, client *Client, deviceID, href string) (*commands.Content, error) {
	retrieveCommand, err := coapconv.NewRetrieveResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, err
	}

	retrievedEvent, err := client.server.raClient.SyncRetrieveResource(req.Context, "*", retrieveCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(retrievedEvent)
	if err != nil {
		return nil, err
	}

	return content, nil
}
