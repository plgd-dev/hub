package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"google.golang.org/grpc/status"
)

// URIToDeviceIDHref convert uri to deviceID and href. Expected input "/oic/route/{deviceID}/{Href}".
func URIToDeviceIDHref(msg *mux.Message) (deviceID, href string, err error) {
	wholePath, err := msg.Options.Path()
	if err != nil {
		return "", "", fmt.Errorf("cannot parse deviceID, href from uri: %w", err)
	}
	path := strings.Split(wholePath, "/")
	if len(path) < 4 {
		return "", "", fmt.Errorf("cannot parse deviceID, href from uri")
	}
	return path[2], fixHref(strings.Join(path[3:], "/")), nil
}

func getResourceInterface(msg *mux.Message) string {
	queries, _ := msg.Options.Queries()
	for _, query := range queries {
		if strings.HasPrefix(query, "if=") {
			return strings.TrimLeft(query, "if=")
		}
	}
	return ""
}

func clientRetrieveHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle retrieve resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}

	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle retrieve resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	var content *commands.Content
	var code coapCodes.Code
	resourceInterface := getResourceInterface(req)
	if resourceInterface == "" {
		content, code, err = clientRetrieveFromResourceShadowHandler(req.Context, client, deviceID, href)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v from resource shadow: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
			return
		}
	} else {
		code = coapCodes.Content
		content, err = clientRetrieveFromDeviceHandler(req, client, deviceID, href)
		if err != nil {
			code = coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Retrieve)
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v from device: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
			return
		}
	}

	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientRetrieveFromResourceShadowHandler(ctx context.Context, client *Client, deviceID, href string) (*commands.Content, coapCodes.Code, error) {
	RetrieveResourcesClient, err := client.server.rdClient.GetResources(ctx, &pbGRPC.GetResourcesRequest{
		ResourceIdFilter: []string{
			commands.NewResourceID(deviceID, href).ToString(),
		},
	})
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Retrieve), err
	}
	defer func() {
		if err := RetrieveResourcesClient.CloseSend(); err != nil {
			log.Errorf("failed to close retrieve devices client: %w", err)
		}
	}()
	for {
		resourceValue, err := RetrieveResourcesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Retrieve), err
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

	retrievedEvent, err := client.server.raClient.SyncRetrieveResource(req.Context, retrieveCommand)
	if err != nil {
		return nil, err
	}
	content, err := commands.EventContentToContent(retrievedEvent)
	if err != nil {
		return nil, err
	}

	return content, nil
}
