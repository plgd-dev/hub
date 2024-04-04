package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	coapMessage "github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
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

func clientRetrieveHandler(req *mux.Message, client *session) (*pool.Message, error) {
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
	etag, err := req.ETag()
	if err != nil {
		etag = nil
	}
	if resourceInterface == "" {
		content, code, err = clientRetrieveFromResourceTwinHandler(req.Context(), client, deviceID, href, etag)
		if err != nil {
			return nil, statusErrorf(code, errFmtRetrieveResource, fmt.Sprintf(" /%v%v from resource twin", deviceID, href), err)
		}
	} else {
		content, code, err = clientRetrieveFromDeviceHandler(req, client, deviceID, href)
		if err != nil {
			code = coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve)
			return nil, statusErrorf(code, errFmtRetrieveResource, fmt.Sprintf(" /%v%v from device", deviceID, href), err)
		}
	}

	if len(content.GetData()) == 0 {
		return client.createResponse(code, req.Token(), coapMessage.TextPlain, nil), nil
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.GetContentType())
	if err != nil {
		return nil, statusErrorf(code, errFmtRetrieveResource, fmt.Sprintf(" /%v%v", deviceID, href), err)
	}
	return client.createResponse(code, req.Token(), mediaType, content.GetData()), nil
}

func clientRetrieveFromResourceTwinHandler(ctx context.Context, client *session, deviceID, href string, etag []byte) (*commands.Content, coapCodes.Code, error) {
	RetrieveResourcesClient, err := client.server.rdClient.GetResources(ctx, &pbGRPC.GetResourcesRequest{
		ResourceIdFilter: []*pbGRPC.ResourceIdFilter{
			{
				ResourceId: commands.NewResourceID(deviceID, href),
			},
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
		if resourceValue.GetData().GetResourceId().GetDeviceId() == deviceID && resourceValue.GetData().GetResourceId().GetHref() == href && resourceValue.GetData().GetContent() != nil {
			if etag != nil && bytes.Equal(etag, resourceValue.GetData().GetEtag()) {
				return nil, coapCodes.Valid, nil
			}
			return resourceValue.GetData().GetContent(), coapCodes.Content, nil
		}
	}
	return nil, coapCodes.NotFound, errors.New("not found")
}

func clientRetrieveFromDeviceHandler(req *mux.Message, client *session, deviceID, href string) (*commands.Content, coapCodes.Code, error) {
	retrieveCommand, err := coapconv.NewRetrieveResourceRequest(commands.NewResourceID(deviceID, href), req, client.RemoteAddr().String())
	if err != nil {
		return nil, coapCodes.ServiceUnavailable, err
	}

	retrievedEvent, err := client.server.raClient.SyncRetrieveResource(req.Context(), "*", retrieveCommand)
	if err != nil {
		return nil, coapCodes.ServiceUnavailable, err
	}
	content, err := commands.EventContentToContent(retrievedEvent)
	if err != nil {
		return nil, coapCodes.ServiceUnavailable, err
	}

	return content, coapconv.StatusToCoapCode(retrievedEvent.GetStatus(), coapconv.Retrieve), nil
}
